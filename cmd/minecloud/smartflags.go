package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/owengage/minecloud/pkg/minecloud"
)

type SmartFlags struct {
	flags *flag.FlagSet

	instanceRequired bool
	worldRequired    bool

	world         *string
	instanceID    *string
	server        *minecloud.MCServer
	acceptNewHost *bool

	mc *minecloud.Minecloud
}

func NewSmartFlags(mc *minecloud.Minecloud, name string) *SmartFlags {
	flags := flag.NewFlagSet(name, flag.ExitOnError)
	acceptNewHost := flags.Bool("accept", false, "accept a new SSH key if found")

	return &SmartFlags{
		flags:         flags,
		acceptNewHost: acceptNewHost,
		mc:            mc,
	}
}

func (f *SmartFlags) InstanceID() string {
	if *f.instanceID == "" {
		panic("instanceID not found, forgot to parse flags?")
	}
	return *f.instanceID
}

func (f *SmartFlags) Server() *minecloud.MCServer {
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

func (f *SmartFlags) ParseValidate(mc *minecloud.Minecloud, args []string) error {
	f.flags.Parse(args)

	if f.instanceRequired {
		if *f.world == "" && *f.instanceID == "" {
			return errors.New("require -world or -instance-id")
		}

		if *f.instanceID == "" {
			server, err := minecloud.FindRunning(mc.EC2, *f.world)
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
		f.mc.Config.SSHDefaultNewKeyBehaviour = minecloud.SSHNewKeyAccept
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
