package minecloud

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// NewAWS makes a new AWS helper object
func NewAWS(sess *session.Session) *AWS {

	return &AWS{
		Session: sess,
		Logger:  logrus.New(),
		EC2:     ec2.New(sess),
		S3:      s3.New(sess),
	}
}

// AWS contains useful bits for working with AWS.
type AWS struct {
	Session *session.Session
	EC2     *ec2.EC2
	S3      *s3.S3
	Logger  *logrus.Logger

	account *string
}

// RunOn runs the given script on the given instance.
func (a *AWS) RunOn(instanceID, script string) error {
	// Need to get the public IP.
	description, err := a.EC2.DescribeInstances(descInput(instanceID))
	if err != nil {
		return err
	}

	if len(description.Reservations) != 1 {
		return errors.New("instance not found")
	}

	if len(description.Reservations[0].Instances) != 1 {
		return errors.New("instance not found")
	}

	ipPtr := description.Reservations[0].Instances[0].PublicIpAddress

	if ipPtr == nil {
		return errors.New("instance has no public IP (terminated?)")
	}

	ip := *ipPtr

	a.Logger.Infof("public IP: %s", ip)

	// TODO get key properly
	key, err := ioutil.ReadFile("/home/ogage/aws/MinecraftServerKeyPair.pem")
	if err != nil {
		return err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("private: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            "ec2-user",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // FIXME: Need public key.
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

	// Only good for CLI, which is why it's in the AWS services object
	// The CLI could have in theory configured this, and something else
	// could configure it differently.
	sshSession.Stdout = os.Stdout
	sshSession.Stderr = os.Stderr

	err = sshSession.Run(script)
	if err != nil {
		return err
	}

	return nil
}

// Account is the AWS account being used to make requests.
func (a *AWS) Account() (string, error) {
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
func (a *AWS) Region() string {
	return *a.Session.Config.Region
}
