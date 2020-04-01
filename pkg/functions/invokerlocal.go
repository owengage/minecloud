package functions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/owengage/minecloud/pkg/awsdetail"
)

type LocalInvoker struct {
	Detail *awsdetail.Detail
}

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
	case "MinecraftSingleton":
		records := Records{}
		err := json.Unmarshal(payload, &records)
		if err != nil {
			return err
		}
		singleton := Singleton{
			Detail:  invoker.Detail,
			Invoker: invoker,
		}
		return singleton.HandleRequest(context.Background(), records)
	default:
		return fmt.Errorf("unknown functions for local invoke: %s", name)
	}
}
