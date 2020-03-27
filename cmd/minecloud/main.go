package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/owengage/minecraft-aws/pkg/minecloud"
	"github.com/sirupsen/logrus"
)

// CLI for minecloud
type CLI struct {
	services *minecloud.Minecloud
	logger   *logrus.Logger
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
	case "remote-setup":
		err = cli.remoteSetup(remainder)
	case "remote-reserve":
		err = cli.remoteReserve(remainder)
	case "remote-bootstrap":
		err = cli.remoteBootstrap(remainder)
	case "remote-download-world":
		err = cli.remoteDownloadWorld(remainder)
	case "remote-start-server":
		err = cli.remoteStartServer(remainder)
	case "remote-status":
		err = cli.remoteStatus(remainder)
	case "remote-stop-server":
		err = cli.remoteStopServer(remainder)
	case "remote-rm-server":
		err = cli.remoteRmServer(remainder)
	case "remote-logs":
		err = cli.remoteLogs(remainder)
	case "remote-upload-world":
		err = cli.remoteUploadWorld(remainder)
	case "aws-account":
		account, err := cli.services.Account()
		if err == nil {
			cli.logger.Infoln(account)
		}
	default:
		err = errors.New("unknown subcommand")
	}

	return err
}

func (cli *CLI) up(args []string) error {
	cmd := flag.NewFlagSet("up", flag.ExitOnError)
	name := cmd.String("world", "", "world name to download")
	cmd.Parse(args)

	if *name == "" {
		return errors.New("-world required")
	}

	server, err := minecloud.FindRunning(cli.services.EC2, *name)
	if err == minecloud.ErrServerNotFound {
		// fine
	} else if err != nil {
		return fmt.Errorf("up: %w", err)
	} else {
		if server.State != "terminated" {
			return fmt.Errorf("up: server already running: %s", server.Name)
		}
	}

	err = minecloud.FindStored(cli.services.S3, *name)
	if err == minecloud.ErrServerNotFound {
		return fmt.Errorf("up: no server called %s, use 'create'", *name)
	} else if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	// Server not running, and we have it in storage. Fire it up!
	err = minecloud.RunStored(cli.services, *name)
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

	for _, server := range servers {
		cli.logger.WithFields(logrus.Fields{
			"name":  server.Name,
			"state": server.State,
		}).Info("Found")
	}

	return nil
}

func (cli *CLI) remoteBootstrap(args []string) error {
	cmd := flag.NewFlagSet("setup-instance", flag.ExitOnError)
	id := cmd.String("instance-id", "", "instance to download world on to")
	cmd.Parse(args)

	if err := validateInstanceID(*id); err != nil {
		return err
	}

	err := minecloud.BootstrapInstance(cli.services, *id)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func (cli *CLI) remoteSetup(args []string) error {
	cmd := flag.NewFlagSet("setup-instance", flag.ExitOnError)
	id := cmd.String("instance-id", "", "instance to download world on to")
	name := cmd.String("world", "", "world name to download")
	cmd.Parse(args)

	if err := validateInstanceID(*id); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("require -world")
	}

	err := minecloud.SetupInstance(cli.services, *id, *name)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func (cli *CLI) remoteReserve(args []string) error {
	cmd := flag.NewFlagSet("remote-reserve", flag.ExitOnError)
	name := cmd.String("world", "", "world name to download")
	cmd.Parse(args)

	if *name == "" {
		return fmt.Errorf("require -world")
	}

	id, err := minecloud.ReserveInstance(cli.services, *name)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	cli.logger.Infof("instance-id: %s", id)

	return nil
}

func (cli *CLI) remoteDownloadWorld(args []string) error {
	cmd := flag.NewFlagSet("remote-download-world", flag.ExitOnError)
	id := cmd.String("instance-id", "", "instance to download world on to")
	name := cmd.String("world", "", "world name to download")
	cmd.Parse(args)

	if err := validateInstanceID(*id); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("require -world")
	}

	err := minecloud.DownloadWorld(cli.services, *id, *name)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func (cli *CLI) remoteUploadWorld(args []string) error {
	cmd := flag.NewFlagSet("remote-upload-world", flag.ExitOnError)
	id := cmd.String("instance-id", "", "instance to upload world from")
	name := cmd.String("world", "", "world name to upload")
	cmd.Parse(args)

	if err := validateInstanceID(*id); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("require -world")
	}

	err := minecloud.UploadWorld(cli.services, *id, *name)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func (cli *CLI) remoteStartServer(args []string) error {
	cmd := flag.NewFlagSet("remote-start-server", flag.ExitOnError)
	id := cmd.String("instance-id", "", "instance to download world on to")
	cmd.Parse(args)

	if err := validateInstanceID(*id); err != nil {
		return err
	}

	err := minecloud.StartServerWrapper(cli.services, *id)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func (cli *CLI) remoteStatus(args []string) error {
	cmd := flag.NewFlagSet("remote-status", flag.ExitOnError)
	id := cmd.String("instance-id", "", "instance to download world on to")
	cmd.Parse(args)

	if err := validateInstanceID(*id); err != nil {
		return err
	}

	resp, err := minecloud.Status(cli.services, *id)
	if err != nil {
		return err
	}

	cli.logger.Info(resp.Status)

	return nil
}

func validateInstanceID(id string) error {
	if id == "" {
		return fmt.Errorf("require -instance-id")
	}
	if !strings.HasPrefix(id, "i-") {
		return errors.New("Instance IDs start with 'i-'")
	}
	return nil
}

func (cli *CLI) remoteStopServer(args []string) error {
	cmd := flag.NewFlagSet("remote-stop-server", flag.ExitOnError)
	id := cmd.String("instance-id", "", "instance to download world on to")
	cmd.Parse(args)

	if err := validateInstanceID(*id); err != nil {
		return err
	}

	err := cli.services.RunOn(*id, "curl -X POST localhost:8080/stop", minecloud.RunOpts{})
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func (cli *CLI) remoteRmServer(args []string) error {
	cmd := flag.NewFlagSet("remote-rm-server", flag.ExitOnError)
	id := cmd.String("instance-id", "", "instance to download world on to")
	cmd.Parse(args)

	if err := validateInstanceID(*id); err != nil {
		return err
	}

	err := cli.services.RunOn(*id, "docker rm -f serverwrapper", minecloud.RunOpts{})
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func (cli *CLI) remoteLogs(args []string) error {
	cmd := flag.NewFlagSet("remote-stop-server", flag.ExitOnError)
	id := cmd.String("instance-id", "", "instance to download world on to")
	cmd.Parse(args)

	if err := validateInstanceID(*id); err != nil {
		return err
	}

	err := cli.services.RunOn(*id, "docker logs serverwrapper", minecloud.RunOpts{})
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func main() {
	logger := logrus.New()

	if len(os.Args) < 2 {
		logger.Fatal("expected subcommand e.g. ls")
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	cli := CLI{
		services: minecloud.NewMinecloud(sess),
		logger:   logger,
	}
	cli.services.Logger = logger

	err := cli.Exec(os.Args)
	if err != nil {
		cli.logger.Fatal(err)
	}
}
