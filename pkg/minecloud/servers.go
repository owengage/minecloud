package minecloud

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"golang.org/x/crypto/ssh"
)

// MCServer is a Minecraft server.
type MCServer struct {
	Name  string
	State string
}

// GetRunning gets the list of current Minecraft servers, including recently terminated.
func GetRunning(svc *ec2.EC2) ([]MCServer, error) {
	serverFilter := &ec2.Filter{
		Name: aws.String("tag-key"),
		Values: []*string{
			aws.String(serverTagKey),
		},
	}

	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{serverFilter},
	})

	if err != nil {
		return nil, err
	}

	servers := []MCServer{}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			servers = append(servers, MCServer{
				Name:  getMCName(instance),
				State: *instance.State.Name,
			})
		}
	}

	return servers, nil
}

func storageKeyForName(name string) string {
	return "servers/" + name + ".tar.gz"
}

// FindStored returns the file name for a servers storage.
// ErrServerNotFound if no file found. Errors if multiple match.
func FindStored(s3Service *s3.S3, name string) error {
	objects, err := s3Service.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s3BucketName),
		Prefix: aws.String("servers/" + name),
	})

	if err != nil {
		return err
	}

	for _, object := range objects.Contents {
		if *object.Key == storageKeyForName(name) {
			return nil // found
		}
	}

	return ErrServerNotFound
}

// RunStored runs a Minecraft server on EC2 from a world stored on S3.
func RunStored(ec2Service *ec2.EC2, s3Service *s3.S3, name string) error {

	reservation, err := ec2Service.RunInstances(&ec2.RunInstancesInput{
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
		ImageId:      aws.String("ami-0cb790308f7591fa6"),
		InstanceType: aws.String("t2.micro"), // FIXME configurable
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String("MinecraftServerRole"),
		},
		TagSpecifications: []*ec2.TagSpecification{
			&ec2.TagSpecification{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					&ec2.Tag{
						Key:   aws.String(serverTagKey),
						Value: aws.String(name),
					},
				},
			},
		},
		SecurityGroupIds: []*string{
			aws.String("sg-001670db09337d6a9"), // FIXME configurable
		},
		KeyName: aws.String("MinecraftServerKeyPair"),
	})

	if err != nil {
		return err
	}

	if len(reservation.Instances) != 1 {
		return fmt.Errorf("runstored: reservation returned non-1 (%d) instances", len(reservation.Instances))
	}

	//instanceID := *reservation.Instances[0].InstanceId

	// err = SetupInstance(ec2Service, s3Service, instanceID)
	// if err != nil {
	// 	return err
	// }

	return nil
}

// SetupInstance sets up an existing EC2 instance into a Minecraft server.
func SetupInstance(services *AWS, instanceID string) error {

	// Get the server to...
	//	- Download the world from S3
	//	- Unpackage or whatever
	//	- Run minecloud/server-wrapper docker container
	// Should then be able to list the server and hopefully access it once ready.

	instanceDescribeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			&instanceID,
		},
	}

	log.Println("waiting for EC2 instance to be running")
	err := services.EC2.WaitUntilInstanceRunning(instanceDescribeInput)
	if err != nil {
		return err
	}

	// Need to get the public IP.
	description, err := services.EC2.DescribeInstances(instanceDescribeInput)
	if err != nil {
		return err
	}

	// FIXME check indexes.
	ip := *description.Reservations[0].Instances[0].PublicIpAddress

	fmt.Println("Public IP:", ip)

	// SSH into box...

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

	sshSess, err := client.NewSession()
	if err != nil {
		return err
	}

	out, err := sshSess.CombinedOutput(`
		sudo yum update -y;
		sudo yum install -y docker;
		sudo service docker start;
		sudo usermod -a -G docker ec2-user;

		# Log in to docker
		# sed hack to remove an invalid argument, god knows why it's there.
		$(aws ecr get-login --region eu-west-2 | sed 's/-e none//g')
	`)

	if err != nil {
		log.Println(string(out))
		return err
	}
	sshSess.Close()

	sshSess, err = client.NewSession()
	if err != nil {
		return err
	}

	out, err = sshSess.CombinedOutput(`
		docker run -d \
			-p 8080:8080 \
			--volume /world:/world \
			344791319371.dkr.ecr.eu-west-2.amazonaws.com/minecloud/server-wrapper:latest \
			-address 0.0.0.0:8080
	`)

	if err != nil {
		log.Println(string(out))
		return err
	}
	sshSess.Close()
	log.Println(string(out))

	// Get the server to...
	//	- Download the world from S3
	//	- Unpackage or whatever
	//	- Run minecloud/server-wrapper docker container
	// Should then be able to list the server and hopefully access it once ready.

	return nil
}

// ErrServerNotFound given if server isn't found on cloud
var ErrServerNotFound error = errors.New("server not found")

// FindRunning returns the server if it exists. Error will be ErrServerNotFound if
// not found, and a different error otherwise.
func FindRunning(svc *ec2.EC2, name string) (MCServer, error) {
	// TODO: Filter AWS request rather than getting all servers.
	servers, err := GetRunning(svc)
	if err != nil {
		return MCServer{}, err
	}

	for _, server := range servers {
		if server.Name == name {
			return server, nil
		}
	}

	return MCServer{}, ErrServerNotFound
}

func getMCName(instance *ec2.Instance) string {
	for _, tag := range instance.Tags {
		if *tag.Key == serverTagKey {
			return *tag.Value
		}
	}
	panic("tried to get server name for instance without tag")
}
