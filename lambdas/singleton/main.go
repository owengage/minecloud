// Goal of this lambda is to quickly send off requests to the MinecraftCommand lambda.
// It will run with only one instance, reading off of a SQS queue, this allows it to do
// atomic checks such as checking if a server is already running a given world.
package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	ls "github.com/aws/aws-sdk-go/service/lambda"

	"github.com/owengage/minecloud/pkg/awsdetail"
	"github.com/owengage/minecloud/pkg/functions"
)

var singleton functions.Singleton

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
		SSHPrivateKey:     functions.GetSSHKey(awsSession),
		SSHKnownHostsPath: "/tmp/known_hosts",
	}

	detail := awsdetail.NewDetail(awsSession, config)

	singleton = functions.Singleton{
		Detail:  detail,
		Invoker: &functions.LambdaInvoker{LS: ls.New(awsSession)},
	}

	lambda.Start(singleton.HandleRequest)
}
