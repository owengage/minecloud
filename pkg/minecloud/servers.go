package minecloud

import (
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
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
func RunStored(services *AWS, name string) error {

	reservation, err := services.EC2.RunInstances(&ec2.RunInstancesInput{
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
		ImageId:      aws.String("ami-0cb790308f7591fa6"),
		InstanceType: aws.String("m5.large"), // FIXME configurable
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

	instanceID := *reservation.Instances[0].InstanceId

	err = SetupInstance(services, instanceID, name)
	if err != nil {
		return err
	}

	return nil
}

// BootstrapInstance takes an existing EC2 instance and installs all prerequisites
// for running a minecraft server.
func BootstrapInstance(services *AWS, instanceID string) error {
	client, err := SSHClient(services, instanceID)
	if err != nil {
		return err
	}
	defer client.Close()

	sshSess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sshSess.Close()

	out, err := sshSess.CombinedOutput(`
		sudo yum update -y;
		sudo yum install -y docker;
		sudo service docker start;
		sudo usermod -a -G docker ec2-user;
	`)

	if err != nil {
		log.Println(string(out))
		return err
	}
	log.Println(string(out))

	return nil
}

// DownloadWorld on remote instance.
func DownloadWorld(services *AWS, instanceID, name string) error {
	client, err := SSHClient(services, instanceID)
	if err != nil {
		return err
	}
	defer client.Close()

	cmd, err := client.NewSession()
	if err != nil {
		return err
	}
	defer cmd.Close()

	s3ObjectPath := "s3://" + s3BucketName + "/" + storageKeyForName(name)

	out, err := cmd.CombinedOutput(fmt.Sprintf(`
		aws s3 cp %s server.tar.gz
		tar xvf server.tar.gz
		rm server.tar.gz
		sudo mv server /server
	`, s3ObjectPath))

	if err != nil {
		log.Println(string(out))
		return err
	}
	log.Println(string(out))

	return nil
}

// StartServerWrapper starts the server wrapper on the EC2 instance that the
// ssh client is connected to. Expects it isn't already running.
func StartServerWrapper(services *AWS, instanceID string) error {
	client, err := SSHClient(services, instanceID)
	if err != nil {
		return err
	}
	defer client.Close()

	sshSess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sshSess.Close()

	region := services.Region()
	account, err := services.Account()
	if err != nil {
		return err
	}

	// TODO template this.
	out, err := sshSess.CombinedOutput(fmt.Sprintf(`
		# Log in to docker
		# sed hack to remove an invalid argument, god knows why it's there.
		$(aws ecr get-login --region %s | sed 's/-e none//g')
		
		docker pull %s.dkr.ecr.%s.amazonaws.com/minecloud/server-wrapper:latest

		docker run -d \
			-p 8080:8080 \
			-p 25565:25565 \
			--volume /server:/server \
			%s.dkr.ecr.%s.amazonaws.com/minecloud/server-wrapper:latest \
			-address 0.0.0.0:8080
	`, region, account, region, account, region))

	if err != nil {
		log.Println(string(out))
		return err
	}
	log.Println(string(out))
	return nil
}

// SetupInstance sets up an existing EC2 instance into a Minecraft server.
func SetupInstance(services *AWS, instanceID, name string) error {

	err := services.EC2.WaitUntilInstanceRunning(descInput(instanceID))
	if err != nil {
		return err
	}

	err = BootstrapInstance(services, instanceID)
	if err != nil {
		return err
	}

	err = DownloadWorld(services, instanceID, name)
	if err != nil {
		return err
	}

	err = StartServerWrapper(services, instanceID)
	if err != nil {
		return err
	}

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

func descInput(instanceID string) *ec2.DescribeInstancesInput {
	return &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			&instanceID,
		},
	}
}
