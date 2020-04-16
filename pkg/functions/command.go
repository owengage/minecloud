package functions

import (
	"context"
	"errors"

	"github.com/owengage/minecloud/pkg/awsdetail"
	"github.com/owengage/minecloud/pkg/minecloud"
)

type Command struct {
	Detail *awsdetail.Detail
}

// HandleRequest from lambda
func (env *Command) HandleRequest(ctx context.Context, event Event) error {
	if event.Command == nil {
		return errors.New("no command specified")
	}

	if event.World == nil {
		return errors.New("no world specified")
	}

	var err error

	switch *event.Command {
	case "up":
		err = awsdetail.RunStored(env.Detail, *event.World)
	case "down":
		err = awsdetail.StoreRunning(env.Detail, *event.World)
	case "backup":
		err = awsdetail.BackupWorld(env.Detail, minecloud.World(*event.World))
	default:
		err = errors.New("unknown command")
	}

	return err
}
