package minecloud

import (
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

// SSHClient gives a SSH client to the EC2 instance.
func SSHClient(services *AWS, instanceID string) (*ssh.Client, error) {

	// Need to get the public IP.
	description, err := services.EC2.DescribeInstances(descInput(instanceID))
	if err != nil {
		return nil, err
	}

	// FIXME check indexes.
	ip := *description.Reservations[0].Instances[0].PublicIpAddress

	fmt.Println("Public IP:", ip)

	// SSH into box...

	// TODO get key properly
	key, err := ioutil.ReadFile("/home/ogage/aws/MinecraftServerKeyPair.pem")
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("private: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            "ec2-user",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // FIXME: Need public key.
	}

	client, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return client, nil
}
