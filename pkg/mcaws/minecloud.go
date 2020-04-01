package mcaws

import (
	"encoding/json"

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
		invoker = &functions.LambdaInvoker{
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

func (a *minecloudAWS) Up(world minecloud.World) error {
	event := functions.Event{
		Command: aws.String("up"),
		World:   aws.String(string(world)),
	}

	eventPayload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	records := functions.Records{
		Records: []functions.Record{
			functions.Record{
				Body: string(eventPayload),
			},
		},
	}

	payload, err := json.Marshal(records)
	if err != nil {
		return err
	}

	return a.invoker.Invoke("MinecloudSingleton", payload)
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

	records := functions.Records{
		Records: []functions.Record{
			functions.Record{
				Body: string(eventPayload),
			},
		},
	}

	payload, err := json.Marshal(records)
	if err != nil {
		return err
	}

	return a.invoker.Invoke("MinecloudSingleton", payload)
}
