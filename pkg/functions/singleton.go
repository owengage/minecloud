// functions implements as much code as possible for the lambda functions
//
// This means we can create a thin wrapper around this code to test our lambda
// functions completely locally.
package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/owengage/minecloud/pkg/awsdetail"
)

type Event struct {
	Command *string `json:"command"`
	World   *string `json:"world"`
}

type Singleton struct {
	Detail  *awsdetail.Detail
	Invoker Invoker
}

type Records struct {
	Records []Record `json:"Records"`
}

type Record struct {
	Body string `json:"body"`
}

// HandleRequest from lambda
func (env *Singleton) HandleRequest(ctx context.Context, records Records) error {
	errorCount := 0

	for _, msg := range records.Records {
		log.Printf("Event: %+v\n", msg.Body)

		event := Event{}
		err := json.Unmarshal([]byte(msg.Body), &event)
		if err != nil {
			errorCount++
			continue
		}

		if event.Command == nil {
			log.Printf("command not specified\n")
			errorCount++
			continue
		}

		if event.World == nil {
			log.Printf("world not specified\n")
			errorCount++
			continue
		}

		switch *event.Command {
		case "up":
			err = env.HandleUp(ctx, event)
		case "down":
			err = nil
		}

		if err != nil {
			fmt.Println(err)
			errorCount++
			continue
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("encounted %d errors", errorCount)
	}

	return nil
}

func (env *Singleton) HandleUp(ctx context.Context, event Event) error {
	err := awsdetail.ClaimWorld(env.Detail, *event.World)
	if err != nil {
		return err
	}

	b, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return env.Invoker.Invoke("MinecraftCommand", b)
}
