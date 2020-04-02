package main

import (
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/owengage/minecloud/pkg/mcaws"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/owengage/minecloud/pkg/awsdetail"
	"github.com/owengage/minecloud/pkg/minecloud"
	"github.com/sirupsen/logrus"
)

// CLI for minecloud
type CLI struct {
	mc     minecloud.Interface
	detail *awsdetail.Detail
	logger *logrus.Logger
}

// Exec based on command line args
func (cli *CLI) Exec(args []string) error {
	subcommand := args[1]
	remainder := args[2:]
	var err error

	switch subcommand {
	// High level commands
	case "up":
		err = cli.up(remainder)
	case "down":
		err = cli.down(remainder)

	// Plumbing up
	case "ls":
		err = cli.ls(remainder)
	case "reserve":
		err = cli.remoteReserve(remainder)
	case "bootstrap":
		err = cli.remoteBootstrap(remainder)
	case "download":
		err = cli.remoteDownloadWorld(remainder)
	case "start":
		err = cli.remoteStartServer(remainder)

	// Diagnostic
	case "status":
		err = cli.remoteStatus(remainder)
	case "logs":
		err = cli.remoteLogs(remainder)
	case "aws-account":
		account, err := cli.detail.Account()
		if err == nil {
			cli.logger.Infoln(account)
		}

	// Plumbing down
	case "upload":
		err = cli.remoteUploadWorld(remainder)
	case "stop":
		err = cli.remoteStopServer(remainder)
	case "kill":
		err = cli.remoteRmServer(remainder)
	case "terminate":
		err = cli.terminate(remainder)

	// Experimental
	case "claim":
		err = cli.debugClaim(remainder)
	case "unclaim":
		err = cli.debugUnclaim(remainder)

	default:
		err = errors.New("unknown subcommand")
	}

	return err
}

func (cli *CLI) up(args []string) error {
	flags := NewSmartFlags(cli.detail, "up").RequireWorld()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return cli.mc.Up(minecloud.World(flags.World()))
}

func (cli *CLI) down(args []string) error {
	flags := NewSmartFlags(cli.detail, "down").RequireInstance().RequireWorld()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return cli.mc.Down(minecloud.World(flags.World()))
}

func (cli *CLI) terminate(args []string) error {
	flags := NewSmartFlags(cli.detail, "terminate").RequireInstance()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return awsdetail.TerminateInstance(cli.detail, flags.InstanceID())
}

func (cli *CLI) ls(args []string) error {

	servers, err := awsdetail.GetRunning(cli.detail.EC2)
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
	flags := NewSmartFlags(cli.detail, "bootstrap").RequireInstance()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return awsdetail.BootstrapInstance(cli.detail, flags.InstanceID())
}

func (cli *CLI) remoteReserve(args []string) error {
	flags := NewSmartFlags(cli.detail, "reserve").RequireWorld()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	id, err := awsdetail.ReserveInstance(cli.detail, flags.World())
	if err != nil {
		return err
	}

	cli.logger.Infof("instance-id: %s", id)

	return nil
}

func (cli *CLI) debugClaim(args []string) error {
	flags := NewSmartFlags(cli.detail, "claim").RequireWorld()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	err := awsdetail.ClaimWorld(cli.detail, flags.World())
	if err != nil {
		return err
	}

	cli.logger.Infof("claimed: %s", flags.World())
	return nil
}

func (cli *CLI) debugUnclaim(args []string) error {
	flags := NewSmartFlags(cli.detail, "unclaim").RequireWorld()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	err := awsdetail.UnclaimWorld(cli.detail, flags.World())
	if err != nil {
		return err
	}

	return nil
}

func (cli *CLI) remoteDownloadWorld(args []string) error {
	flags := NewSmartFlags(cli.detail, "download-world").RequireInstance().RequireWorld()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return awsdetail.DownloadWorld(cli.detail, flags.InstanceID(), flags.World())
}

func (cli *CLI) remoteUploadWorld(args []string) error {
	flags := NewSmartFlags(cli.detail, "upload-world").RequireInstance().RequireWorld()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return awsdetail.UploadWorld(cli.detail, flags.InstanceID(), flags.World())
}

func (cli *CLI) remoteStartServer(args []string) error {
	flags := NewSmartFlags(cli.detail, "status").RequireInstance()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return awsdetail.StartServerWrapper(cli.detail, flags.InstanceID())
}

func (cli *CLI) remoteStatus(args []string) error {
	flags := NewSmartFlags(cli.detail, "status").RequireInstance()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	resp, err := awsdetail.Status(cli.detail, flags.InstanceID())
	if err != nil {
		return err
	}

	cli.logger.Info(resp.Status)

	return nil
}

func (cli *CLI) remoteStopServer(args []string) error {
	flags := NewSmartFlags(cli.detail, "stop-server").RequireInstance()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return cli.detail.RunOn(flags.InstanceID(), "curl -X POST localhost:8080/stop", awsdetail.RunOpts{})
}

func (cli *CLI) remoteRmServer(args []string) error {
	flags := NewSmartFlags(cli.detail, "rm-server").RequireInstance()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return cli.detail.RunOn(flags.InstanceID(), "docker rm -f serverwrapper", awsdetail.RunOpts{})
}

func (cli *CLI) remoteLogs(args []string) error {
	flags := NewSmartFlags(cli.detail, "logs").RequireInstance()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return cli.detail.RunOn(flags.InstanceID(), "docker logs serverwrapper", awsdetail.RunOpts{})
}

func main() {
	logger := logrus.New()

	if len(os.Args) < 2 {
		logger.Fatal("expected subcommand e.g. ls")
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	home := os.Getenv("HOME")

	config := awsdetail.Config{
		SSHPrivateKeyFile: path.Join(home, ".minecloud", "MinecraftServerKeyPair.pem"),
		SSHKnownHostsPath: path.Join(home, ".ssh/known_hosts"),
	}

	detail := awsdetail.NewDetail(sess, config)
	mc := mcaws.NewMinecloudAWS(sess, detail, true)

	cli := CLI{
		mc:     mc,
		detail: detail,
		logger: logger,
	}
	cli.detail.Logger = logger

	err := cli.Exec(os.Args)
	if err != nil {
		cli.logger.Fatal(err)
	}
}
