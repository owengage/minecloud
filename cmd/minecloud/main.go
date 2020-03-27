package main

import (
	"errors"
	"fmt"
	"os"

	"encoding/json"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/owengage/minecloud/pkg/minecloud"
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
	// High level commands
	case "ls":
		err = cli.ls(remainder)
	case "up":
		err = cli.up(remainder)
	case "down":
		err = cli.down(remainder)

	// Plumbing up
	case "remote-reserve":
		err = cli.remoteReserve(remainder)
	case "remote-bootstrap":
		err = cli.remoteBootstrap(remainder)
	case "remote-download-world":
		err = cli.remoteDownloadWorld(remainder)
	case "remote-start-server":
		err = cli.remoteStartServer(remainder)

	// Diagnostic
	case "remote-status":
		err = cli.remoteStatus(remainder)
	case "remote-logs":
		err = cli.remoteLogs(remainder)
	case "aws-account":
		account, err := cli.services.Account()
		if err == nil {
			cli.logger.Infoln(account)
		}

	// Plumbing down
	case "remote-upload-world":
		err = cli.remoteUploadWorld(remainder)
	case "remote-stop-server":
		err = cli.remoteStopServer(remainder)
	case "remote-rm-server":
		err = cli.remoteRmServer(remainder)
	case "terminate":
		err = cli.terminate(remainder)

	default:
		err = errors.New("unknown subcommand")
	}

	return err
}

func (cli *CLI) up(args []string) error {
	flags := NewSmartFlags("bootstrap").RequireWorld()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	err := minecloud.FindStored(cli.services.S3, flags.World())
	if err == minecloud.ErrServerNotFound {
		return fmt.Errorf("up: no server called %s, use 'create'", flags.World())
	} else if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	// Server not running, and we have it in storage. Fire it up!
	return minecloud.RunStored(cli.services, flags.World())
}

func (cli *CLI) down(args []string) error {
	flags := NewSmartFlags("bootstrap").RequireInstance().RequireWorld()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	server := flags.Server()

	if server.InstanceState == "terminated" || server.InstanceState == "shutting-down" {
		return fmt.Errorf("down: server already terminated: %s", server.Name)
	}

	err := minecloud.StopServerWrapper(cli.services, server.InstanceID)
	if err != nil {
		return fmt.Errorf("down: failed to stop server wrapper (%s): %w", server.Name, err)
	}

	err = minecloud.UploadWorld(cli.services, server.InstanceID, server.Name)
	if err != nil {
		return fmt.Errorf("down: failed to upload world (%s): %w", server.Name, err)
	}

	return minecloud.TerminateInstance(cli.services, server.InstanceID)
}

func (cli *CLI) terminate(args []string) error {
	flags := NewSmartFlags("terminate").RequireInstance()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	return minecloud.TerminateInstance(cli.services, flags.InstanceID())
}

func (cli *CLI) ls(args []string) error {

	servers, err := minecloud.GetRunning(cli.services.EC2)
	if err != nil {
		return err
	}

	for _, server := range servers {
		j, err := json.Marshal(server)
		if err != nil {
			return err
		}

		cli.logger.Infof("%s", j)
	}

	return nil
}

func (cli *CLI) remoteBootstrap(args []string) error {
	flags := NewSmartFlags("bootstrap").RequireInstance()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	return minecloud.BootstrapInstance(cli.services, flags.InstanceID())
}

func (cli *CLI) remoteReserve(args []string) error {
	flags := NewSmartFlags("reserve").RequireWorld()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	id, err := minecloud.ReserveInstance(cli.services, flags.World())
	if err != nil {
		return err
	}

	cli.logger.Infof("instance-id: %s", id)

	return nil
}

func (cli *CLI) remoteDownloadWorld(args []string) error {
	flags := NewSmartFlags("download-world").RequireInstance().RequireWorld()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	return minecloud.DownloadWorld(cli.services, flags.InstanceID(), flags.World())
}

func (cli *CLI) remoteUploadWorld(args []string) error {
	flags := NewSmartFlags("upload-world").RequireInstance().RequireWorld()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	return minecloud.UploadWorld(cli.services, flags.InstanceID(), flags.World())
}

func (cli *CLI) remoteStartServer(args []string) error {
	flags := NewSmartFlags("status").RequireInstance()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	return minecloud.StartServerWrapper(cli.services, flags.InstanceID())
}

func (cli *CLI) remoteStatus(args []string) error {
	flags := NewSmartFlags("status").RequireInstance()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	resp, err := minecloud.Status(cli.services, flags.InstanceID())
	if err != nil {
		return err
	}

	cli.logger.Info(resp.Status)

	return nil
}

func (cli *CLI) remoteStopServer(args []string) error {
	flags := NewSmartFlags("stop-server").RequireInstance()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	return cli.services.RunOn(flags.InstanceID(), "curl -X POST localhost:8080/stop", minecloud.RunOpts{})
}

func (cli *CLI) remoteRmServer(args []string) error {
	flags := NewSmartFlags("rm-server").RequireInstance()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	return cli.services.RunOn(flags.InstanceID(), "docker rm -f serverwrapper", minecloud.RunOpts{})
}

func (cli *CLI) remoteLogs(args []string) error {
	flags := NewSmartFlags("logs").RequireInstance()
	if err := flags.ParseValidate(cli.services, args); err != nil {
		return err
	}

	return cli.services.RunOn(flags.InstanceID(), "docker logs serverwrapper", minecloud.RunOpts{})
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
