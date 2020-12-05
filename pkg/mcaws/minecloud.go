package mcaws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/owengage/minecloud/pkg/awsdetail"
	"github.com/owengage/minecloud/pkg/functions"
	"github.com/owengage/minecloud/pkg/minecloud"
)

func NewMinecloudAWS(sess *session.Session, detail *awsdetail.Detail, localLambda bool) minecloud.Interface {
	var invoker functions.Invoker

	if localLambda {
		invoker = &functions.LocalInvoker{
			Detail: detail,
		}
	} else {
		invoker = &awsdetail.LambdaInvoker{
			LS: lambda.New(sess),
		}
	}

	return &minecloudAWS{
		detail:  detail,
		invoker: invoker,
	}
}

type minecloudAWS struct {
	detail  *awsdetail.Detail
	invoker functions.Invoker
}

func (a *minecloudAWS) Up(world minecloud.World, instanceType *string) error {
	event := functions.Event{
		Command:      aws.String("up"),
		World:        aws.String(string(world)),
		InstanceType: instanceType,
	}

	fmt.Printf("event going in: %+v\n", event)
	eventPayload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return a.invoker.Invoke("MinecloudSingleton", eventPayload)
}

func (a *minecloudAWS) Down(world minecloud.World) error {
	event := functions.Event{
		Command: aws.String("down"),
		World:   aws.String(string(world)),
	}

	eventPayload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return a.invoker.Invoke("MinecloudSingleton", eventPayload)
}
