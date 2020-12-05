package awsdetail

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

// NewDetail makes detail.new AWS helper object
func NewDetail(sess *session.Session, config Config) *Detail {
	if config.SSHDefaultNewKeyBehaviour == SSHNewKeyUnspecified {
		config.SSHDefaultNewKeyBehaviour = SSHNewKeyReject
	}

	return &Detail{
		Session: sess,
		Logger:  logrus.New(),
		EC2:     ec2.New(sess),
		S3:      s3.New(sess),
		Config:  config,
	}
}

// Detail contains useful bits for working with AWS.
type Detail struct {
	Session *session.Session
	EC2     *ec2.EC2
	S3      *s3.S3
	Logger  *logrus.Logger
	Config  Config

	account *string
}

// SSHNewKeyOpt indicates how to treat unknown hosts with SSH.
type SSHNewKeyOpt int

const (
	// SSHNewKeyUnspecified default value placeholder.
	SSHNewKeyUnspecified SSHNewKeyOpt = iota

	// SSHNewKeyReject reject any unknown key.
	SSHNewKeyReject

	// SSHNewKeyAccept accept detail. unknown key. Acceptable for first-contact only.
	SSHNewKeyAccept
)

// Config for AWS Detail.
type Config struct {
	SSHPrivateKey             []byte
	SSHPrivateKeyFile         string
	SSHKnownHostsPath         string
	SSHDefaultNewKeyBehaviour SSHNewKeyOpt
	HostedZoneID              string
	HostedZoneSuffix          string // eg "example.com." note final dot.
}

// RunOpts options when running commands tunnelling through SSH.
type RunOpts struct {
	Stdout          io.Writer
	Stderr          io.Writer
	NewKeyBehaviour SSHNewKeyOpt
}

// RunOn runs the given script on the given instance.
func (detail *Detail) RunOn(instanceID, script string, opts RunOpts) error {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	return detail.runOn(instanceID, script, opts)
}

// OutputOn returns stdout of running the given script
func (detail *Detail) OutputOn(instanceID, script string, opts RunOpts) ([]byte, []byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	opts.Stdout = &stdout
	opts.Stderr = &stderr

	err := detail.runOn(instanceID, script, opts)
	return stdout.Bytes(), stderr.Bytes(), err
}

// Account is the AWS account being used to make requests.
func (detail *Detail) Account() (string, error) {
	if detail.account == nil {
		STS := sts.New(detail.Session)
		identity, err := STS.GetCallerIdentity(nil)
		if err != nil {
			return "", err
		}

		detail.account = identity.Account
	}

	return *detail.account, nil
}

// Region returns the region we are running commands in.
func (detail *Detail) Region() string {
	return *detail.Session.Config.Region
}

// IP gets the IP of an instance.
func (detail *Detail) IP(instanceID string) (string, error) {
	description, err := detail.EC2.DescribeInstances(descInput(instanceID))
	if err != nil {
		return "", err
	}

	if len(description.Reservations) != 1 {
		return "", fmt.Errorf("instance not found (%d reservations)", len(description.Reservations))
	}

	if len(description.Reservations[0].Instances) != 1 {
		return "", fmt.Errorf("instance not found (%d instances in reservation)", len(description.Reservations[0].Instances))
	}

	ipPtr := description.Reservations[0].Instances[0].PublicIpAddress

	if ipPtr == nil {
		return "", errors.New("instance has no public IP (terminated?)")
	}

	return *ipPtr, nil
}

// runOn runs the given script on the given instance.
func (detail *Detail) runOn(instanceID, script string, opts RunOpts) error {
	err := ensureKeyBytes(detail)
	if err != nil {
		return err
	}

	if opts.NewKeyBehaviour == SSHNewKeyUnspecified {
		opts.NewKeyBehaviour = detail.Config.SSHDefaultNewKeyBehaviour
	}

	ip, err := detail.IP(instanceID)

	if err != nil {
		return errors.New("instance has no public IP (terminated?)")
	}

	signer, err := ssh.ParsePrivateKey(detail.Config.SSHPrivateKey)
	if err != nil {
		return fmt.Errorf("private: %w", err)
	}

	hostCallback := detail.tofuCallback(opts)

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

// tofuCallback creates detail.new host callback for SSH that can trust on first use *iff* accept new key is true.
func (detail *Detail) tofuCallback(opts RunOpts) func(hostname string, remote net.Addr, key ssh.PublicKey) error {
	knownHostsFile := detail.Config.SSHKnownHostsPath

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

		// If not in hosts but we accept detail.new key, add the key to the hosts file.
		if opts.NewKeyBehaviour == SSHNewKeyAccept {
			detail.Logger.Info("adding new host to known_hosts")

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

func ensureKeyBytes(detail *Detail) error {
	if detail.Config.SSHPrivateKey != nil {
		return nil
	}

	key, err := ioutil.ReadFile(detail.Config.SSHPrivateKeyFile)
	if err != nil {
		return err
	}

	detail.Config.SSHPrivateKey = key
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
