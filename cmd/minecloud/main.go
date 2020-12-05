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

	cmdMap := map[string]func([]string) error{
		// high level commands
		"up":   cli.up,
		"down": cli.down,

		// plumbing commands
		"init":       cli.init,
		"ls":         cli.ls,
		"reserve":    cli.remoteReserve,
		"bootstrap":  cli.remoteBootstrap,
		"download":   cli.remoteDownloadWorld,
		"start":      cli.remoteStartServer,
		"status":     cli.remoteStatus,
		"logs":       cli.remoteLogs,
		"upload":     cli.remoteUploadWorld,
		"stop":       cli.remoteStopServer,
		"kill":       cli.remoteRmServer,
		"terminate":  cli.terminate,
		"claim":      cli.debugClaim,
		"unclaim":    cli.debugUnclaim,
		"update-dns": cli.updateDNS,
		"save":       cli.save,
		"aws-account": func(remainder []string) error {
			account, err := cli.detail.Account()
			if err == nil {
				cli.logger.Infoln(account)
			}
			return err
		},
	}

	f, ok := cmdMap[subcommand]
	if !ok {
		return errors.New("unknown subcommand")
	}
	return f(remainder)
}

func (cli *CLI) up(args []string) error {
	flags := NewSmartFlags(cli.detail, "up").RequireWorld().RequireInstanceType()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	return cli.mc.Up(minecloud.World(flags.World()), flags.InstanceType())
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

func (cli *CLI) init(args []string) error {
	return awsdetail.Init(cli.detail)
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
	flags := NewSmartFlags(cli.detail, "reserve").RequireWorld().RequireInstanceType()
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	id, err := awsdetail.ReserveInstance(cli.detail, flags.World(), flags.InstanceType())
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

func (cli *CLI) updateDNS(args []string) error {
	flags := NewSmartFlags(cli.detail, "update-dns").RequireWorld()
	ip := flags.flags.String("ip", "", "IP address to point DNS record to")
	if err := flags.ParseValidate(cli.detail, args); err != nil {
		return err
	}

	world := flags.World()

	return awsdetail.UpdateDNS(cli.detail, *ip, minecloud.World(world))
}

func (cli *CLI) save(args []string) error {
	return errors.New("not implemented")
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
		HostedZoneID:      "Z0259601KLGA9PWJ5S0",
		HostedZoneSuffix:  "owengage.com.",
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
