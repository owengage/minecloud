package functions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/owengage/minecloud/pkg/awsdetail"
)

// LocalInvoker invokes the same code as AWS lambdas, but locally.
// Note that the functions themselves may still interact with AWS.
type LocalInvoker struct {
	Detail *awsdetail.Detail
}

// Invoke function locally.
func (invoker *LocalInvoker) Invoke(name string, payload []byte) error {
	switch name {
	case "MinecraftCommand":
		event := Event{}
		err := json.Unmarshal(payload, &event)
		if err != nil {
			return err
		}
		cmd := Command{Detail: invoker.Detail}
		return cmd.HandleRequest(context.Background(), event)
	case "MinecloudSingleton":
		event := Event{}
		err := json.Unmarshal(payload, &event)
		if err != nil {
			return err
		}
		singleton := Singleton{
			Detail:  invoker.Detail,
			Invoker: invoker,
		}
		return singleton.HandleRequest(context.Background(), event)
	default:
		return fmt.Errorf("unknown functions for local invoke: %s", name)
	}
}
