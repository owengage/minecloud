package minecloud

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// NewMinecloud makes a new AWS helper object
func NewMinecloud(sess *session.Session, config Config) *Minecloud {
	if config.SSHDefaultNewKeyBehaviour == SSHNewKeyUnspecified {
		config.SSHDefaultNewKeyBehaviour = SSHNewKeyReject
	}

	return &Minecloud{
		Session: sess,
		Logger:  logrus.New(),
		EC2:     ec2.New(sess),
		S3:      s3.New(sess),
		Config:  config,
	}
}

// Minecloud contains useful bits for working with AWS.
type Minecloud struct {
	Session *session.Session
	EC2     *ec2.EC2
	S3      *s3.S3
	Logger  *logrus.Logger
	Config  Config

	account *string
}

type SSHNewKeyOpt int

const (
	SSHNewKeyUnspecified SSHNewKeyOpt = iota
	SSHNewKeyReject
	SSHNewKeyAccept
)

type Config struct {
	SSHPrivateKey             []byte
	SSHPrivateKeyFile         string
	SSHKnownHostsPath         string
	SSHDefaultNewKeyBehaviour SSHNewKeyOpt
}

type RunOpts struct {
	Stdout          io.Writer
	Stderr          io.Writer
	NewKeyBehaviour SSHNewKeyOpt
}

// RunOn runs the given script on the given instance.
func (a *Minecloud) RunOn(instanceID, script string, opts RunOpts) error {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	return a.runOn(instanceID, script, opts)
}

// OutputOn returns stdout of running the given script
func (a *Minecloud) OutputOn(instanceID, script string, opts RunOpts) ([]byte, []byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	opts.Stdout = &stdout
	opts.Stderr = &stderr

	err := a.runOn(instanceID, script, opts)
	return stdout.Bytes(), stderr.Bytes(), err
}

// Account is the AWS account being used to make requests.
func (a *Minecloud) Account() (string, error) {
	if a.account == nil {
		STS := sts.New(a.Session)
		identity, err := STS.GetCallerIdentity(nil)
		if err != nil {
			return "", err
		}

		a.account = identity.Account
	}

	return *a.account, nil
}

// Region returns the region we are running commands in.
func (a *Minecloud) Region() string {
	return *a.Session.Config.Region
}

// runOn runs the given script on the given instance.
func (a *Minecloud) runOn(instanceID, script string, opts RunOpts) error {
	// Need to get the public IP.
	description, err := a.EC2.DescribeInstances(descInput(instanceID))
	if err != nil {
		return err
	}

	err = ensureKeyBytes(a)
	if err != nil {
		return err
	}

	if opts.NewKeyBehaviour == SSHNewKeyUnspecified {
		opts.NewKeyBehaviour = a.Config.SSHDefaultNewKeyBehaviour
	}

	if len(description.Reservations) != 1 {
		return fmt.Errorf("instance not found (%d reservations)",
			len(description.Reservations))
	}

	if len(description.Reservations[0].Instances) != 1 {
		return fmt.Errorf("instance not found (%d instances in reservation)",
			len(description.Reservations[0].Instances))
	}

	ipPtr := description.Reservations[0].Instances[0].PublicIpAddress

	if ipPtr == nil {
		return errors.New("instance has no public IP (terminated?)")
	}

	ip := *ipPtr

	signer, err := ssh.ParsePrivateKey(a.Config.SSHPrivateKey)
	if err != nil {
		return fmt.Errorf("private: %w", err)
	}

	hostCallback := a.tofuCallback(opts)

	config := &ssh.ClientConfig{
		User:            "ec2-user",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostCallback,
	}

	client, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer client.Close()

	sshSession, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sshSession.Close()

	sshSession.Stdout = opts.Stdout
	sshSession.Stderr = opts.Stderr

	err = sshSession.Run(script)
	if err != nil {
		return err
	}

	return nil
}

func (mc *Minecloud) tofuCallback(opts RunOpts) func(hostname string, remote net.Addr, key ssh.PublicKey) error {
	knownHostsFile := mc.Config.SSHKnownHostsPath

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		hostKeyCallback, err := knownhosts.New(knownHostsFile)
		if err != nil {
			return fmt.Errorf("could not create hostkeycallback function: %w", err)
		}

		// If we're in the known hosts, happy days
		err = hostKeyCallback(hostname, remote, key)
		if err == nil {
			return nil
		}

		// If not in hosts but we accept a new key, add the key to the hosts file.
		if opts.NewKeyBehaviour == SSHNewKeyAccept {
			mc.Logger.Info("adding new host to known_hosts")

			err = addToKnownHosts(knownHostsFile, hostname, key)
			if err != nil {
				return err
			}
		}

		// try hosts file again now we've added it, just to verify.
		hostKeyCallback, err = knownhosts.New(knownHostsFile)
		if err != nil {
			return fmt.Errorf("could not create hostkeycallback function: %w", err)
		}

		return hostKeyCallback(hostname, remote, key)
	}
}

func ensureKeyBytes(mc *Minecloud) error {
	if mc.Config.SSHPrivateKey != nil {
		return nil
	}

	key, err := ioutil.ReadFile(mc.Config.SSHPrivateKeyFile)
	if err != nil {
		return err
	}

	mc.Config.SSHPrivateKey = key
	return nil
}

func addToKnownHosts(knownHostsFile, hostname string, key ssh.PublicKey) error {
	hostname = knownhosts.Normalize(hostname)
	line := knownhosts.Line([]string{hostname}, key)

	file, err := os.OpenFile(knownHostsFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte(line + "\n"))
	return err
}
