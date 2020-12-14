// functions implements as much code as possible for the lambda functions
//
// This means we can create a thin wrapper around this code to test our lambda
// functions completely locally.
package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/owengage/minecloud/pkg/awsdetail"
)

type Event struct {
	Command      *string `json:"command"`
	World        *string `json:"world"`
	InstanceType *string `json:"instanceType"`
}

type Singleton struct {
	Detail  *awsdetail.Detail
	Invoker Invoker
}

// HandleRequest from lambda
func (env *Singleton) HandleRequest(ctx context.Context, event Event) error {

	envEventPayload := os.Getenv("EVENT")

	if envEventPayload != "" {
		// We have a payload via environment variable. Use this as the event
		// instead. Useful for when the lambda is triggered via something like
		// CloudWatch events. We can create a lambda that shares the same
		// function code, but changes this EVENT environment variable.
		json.Unmarshal([]byte(envEventPayload), &event)
	}

	if event.Command == nil {
		return fmt.Errorf("command not specified")
	}

	if event.World == nil {
		return fmt.Errorf("world not specified")

	}

	switch *event.Command {
	case "up":
		return env.HandleUp(ctx, event)
	case "down":
		return env.HandleDown(ctx, event)
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

func (env *Singleton) HandleDown(ctx context.Context, event Event) error {
	// TODO: Add an "isClaimed" type check.

	b, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return env.Invoker.Invoke("MinecraftCommand", b)
}
