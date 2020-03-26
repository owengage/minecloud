package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/owengage/minecraft-aws/pkg/minecloud"
)

type CLI struct {
	services *minecloud.AWS
}

// Exec based on command line args
func (cli *CLI) Exec(args []string) error {
	subcommand := args[1]
	remainder := args[2:]
	var err error

	switch subcommand {
	case "ls":
		err = cli.ls(remainder)
	case "up":
		err = cli.up(remainder)
	case "run-instance":
		err = cli.runInstance(remainder)
	case "setup-instance":
		err = cli.setupInstance(remainder)
	case "aws-account":
		account, err := cli.services.Account()
		if err == nil {
			fmt.Println(account)
		}
	}

	return err
}

func (cli *CLI) up(args []string) error {
	cmd := flag.NewFlagSet("up", flag.ExitOnError)
	cmd.Parse(args)
	if cmd.NArg() != 1 {
		return fmt.Errorf("require server name")
	}

	name := cmd.Arg(0)

	server, err := minecloud.FindRunning(cli.services.EC2, name)
	if err == minecloud.ErrServerNotFound {
		// fine
	} else if err != nil {
		return fmt.Errorf("up: %w", err)
	} else {
		if server.State != "terminated" {
			return fmt.Errorf("up: server already running: %s", server.Name)
		}
	}

	err = minecloud.FindStored(cli.services.S3, name)
	if err == minecloud.ErrServerNotFound {
		return fmt.Errorf("up: no server called %s, use 'create'", name)
	} else if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	// Server not running, and we have it in storage. Fire it up!
	err = minecloud.RunStored(cli.services.EC2, cli.services.S3, name)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func (cli *CLI) ls(args []string) error {
	cmd := flag.NewFlagSet("ls", flag.ExitOnError)
	cmd.Parse(args)
	if len(cmd.Args()) != 0 {
		return fmt.Errorf("too many arguments to ls")
	}

	servers, err := minecloud.GetRunning(cli.services.EC2)
	if err != nil {
		return err
	}

	fmt.Printf("%s\t%s\n", "NAME", "STATE")
	for _, server := range servers {
		fmt.Printf("%s\t%s\n", server.Name, server.State)
	}

	return nil
}

func (cli *CLI) runInstance(args []string) error {
	cmd := flag.NewFlagSet("run-instance", flag.ExitOnError)
	cmd.Parse(args)
	if cmd.NArg() != 1 {
		return fmt.Errorf("require server name")
	}

	name := cmd.Arg(0)

	err := minecloud.RunStored(cli.services.EC2, cli.services.S3, name)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func (cli *CLI) setupInstance(args []string) error {
	cmd := flag.NewFlagSet("setup-instance", flag.ExitOnError)
	cmd.Parse(args)
	if cmd.NArg() != 1 {
		return fmt.Errorf("require instance ID")
	}

	id := cmd.Arg(0)
	if !strings.HasPrefix(id, "i-") {
		return errors.New("Instance IDs start with 'i-'")
	}

	err := minecloud.SetupInstance(cli.services, id)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("expected subcommand e.g. ls")
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	cli := CLI{
		services: minecloud.NewAWS(sess),
	}

	err := cli.Exec(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
