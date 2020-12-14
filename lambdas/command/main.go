package main

import (
	"os"

	"github.com/owengage/minecloud/pkg/awsdetail"
	"github.com/owengage/minecloud/pkg/functions"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
)

var detail *awsdetail.Detail

func main() {
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Touch the hosts file to make sure it exists.
	f, err := os.OpenFile("/tmp/known_hosts", os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	f.Close()

	config := awsdetail.Config{
		SSHPrivateKey:             functions.GetSSHKey(awsSession),
		SSHKnownHostsPath:         "/tmp/known_hosts",
		SSHDefaultNewKeyBehaviour: awsdetail.SSHNewKeyAccept,
		HostedZoneID:              "Z0259601KLGA9PWJ5S0",
		HostedZoneSuffix:          "owengage.com.",
	}

	detail = awsdetail.NewDetail(awsSession, config)

	cmd := functions.Command{
		Detail: detail,
	}

	lambda.Start(cmd.HandleRequest)
}
