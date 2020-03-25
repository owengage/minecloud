package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/owengage/minecraft-aws/pkg/minecloud"
)

func lsCmd(session *session.Session, args []string) error {
	cmd := flag.NewFlagSet("ls", flag.ExitOnError)
	cmd.Parse(args)
	if len(cmd.Args()) != 0 {
		return fmt.Errorf("too many arguments to ls")
	}

	svc := ec2.New(session)
	servers, err := minecloud.GetRunning(svc)

	if err != nil {
		return err
	}

	fmt.Printf("%s\t%s\n", "NAME", "STATE")
	for _, server := range servers {
		fmt.Printf("%s\t%s\n", server.Name, server.State)
	}

	return nil
}

func upCmd(session *session.Session, args []string) error {
	cmd := flag.NewFlagSet("up", flag.ExitOnError)
	cmd.Parse(args)
	if cmd.NArg() != 1 {
		return fmt.Errorf("require server name")
	}

	name := cmd.Arg(0)

	// Want to check that the server isn't already running
	ec2Service := ec2.New(session)

	server, err := minecloud.FindRunning(ec2Service, name)
	if err == minecloud.ErrServerNotFound {
		// fine
	} else if err != nil {
		return fmt.Errorf("up: %w", err)
	} else {
		return fmt.Errorf("up: server already running: %s", server.Name)
	}

	// Server not running, but do we actually have a server with this name?
	s3Service := s3.New(session)

	err = minecloud.FindStored(s3Service, name)
	if err == minecloud.ErrServerNotFound {
		return fmt.Errorf("up: no server called %s, use 'create'", name)
	} else if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	log.Printf("found server storage for %s", name)
	// Server not running, and we have it in storage. Fire it up!
	// _, err = minecloud.RunStored(ec2Service, s3Service, name)
	// if err != nil {
	// 	return fmt.Errorf("up: %w", err)
	// }

	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("expected subcommand e.g. ls")
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	var err error
	args := os.Args[2:]

	switch os.Args[1] {
	case "ls":
		err = lsCmd(sess, args)
	case "up":
		err = upCmd(sess, args)
	}

	if err != nil {
		log.Fatal(err)
	}
}
