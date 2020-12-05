package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/owengage/minecloud/pkg/awsdetail"
)

type SmartFlags struct {
	flags *flag.FlagSet

	instanceRequired bool
	worldRequired    bool

	world         *string
	instanceID    *string
	instanceType  *string
	server        *awsdetail.MCServer
	acceptNewHost *bool

	detail *awsdetail.Detail
}

func NewSmartFlags(detail *awsdetail.Detail, name string) *SmartFlags {
	flags := flag.NewFlagSet(name, flag.ExitOnError)
	acceptNewHost := flags.Bool("accept", false, "accept a new SSH key if found")

	return &SmartFlags{
		flags:         flags,
		acceptNewHost: acceptNewHost,
		detail:        detail,
	}
}

func (f *SmartFlags) InstanceID() string {
	if *f.instanceID == "" {
		panic("instanceID not found, forgot to parse flags?")
	}
	return *f.instanceID
}

func (f *SmartFlags) Server() *awsdetail.MCServer {
	if f.server == nil {
		panic("server not found, forgot to parse flags?")
	}
	return f.server
}

func (f *SmartFlags) World() string {
	if *f.world == "" {
		panic("world not found, forgot to parse flags?")
	}
	return *f.world
}

func (f *SmartFlags) InstanceType() *string {
	// Stupid flag package sets the flag to the default value
	// rather than nil. Detect that and change it here to not polute
	// that concept to the rest of the application.
	if *f.instanceType != "" {
		return f.instanceType
	}
	return nil
}

func (f *SmartFlags) RequireInstance() *SmartFlags {
	if f.world == nil {
		f.world = f.flags.String("world", "", "name of world")
	}

	f.instanceID = f.flags.String("instance-id", "", "instance ID of EC2 instance")
	f.instanceRequired = true
	return f
}

func (f *SmartFlags) RequireWorld() *SmartFlags {
	if f.world == nil {
		f.world = f.flags.String("world", "", "name of world")
	}
	f.worldRequired = true
	return f
}

func (f *SmartFlags) RequireInstanceType() *SmartFlags {
	if f.instanceType == nil {
		f.instanceType = f.flags.String("instance-type", "", "type of EC2 instance to use")
	}
	return f
}

func (f *SmartFlags) ParseValidate(detail *awsdetail.Detail, args []string) error {
	f.flags.Parse(args)

	if f.instanceRequired {
		if *f.world == "" && *f.instanceID == "" {
			return errors.New("require -world or -instance-id")
		}

		if *f.instanceID == "" {
			server, err := awsdetail.FindRunning(detail.EC2, *f.world)
			if err != nil {
				return err
			}
			*f.instanceID = server.InstanceID
			f.server = &server

			if err := validateInstanceID(*f.instanceID); err != nil {
				return err
			}
		}
	}

	if f.worldRequired {
		if *f.world == "" {
			return errors.New("require -world")
		}
	}

	if *f.acceptNewHost {
		f.detail.Config.SSHDefaultNewKeyBehaviour = awsdetail.SSHNewKeyAccept
	}

	return nil
}

func validateInstanceID(id string) error {
	if id == "" {
		return fmt.Errorf("require -instance-id")
	}
	if !strings.HasPrefix(id, "i-") {
		return errors.New("Instance IDs start with 'i-'")
	}
	return nil
}
