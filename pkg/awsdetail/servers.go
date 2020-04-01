package awsdetail

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/owengage/minecloud/pkg/serverwrapper"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
)

// ErrServerNotFound given if server isn't found on cloud
var ErrServerNotFound error = errors.New("server not found")

// ErrWorldAlreadyClaimed given if a world is already claimed for a server.
var ErrWorldAlreadyClaimed error = errors.New("world already claimed")

// ErrWorldNotClaimed given if a world is already NOT claimed for a server.
var ErrWorldNotClaimed error = errors.New("world already not claimed")

// MCServer is a Minecraft server.
type MCServer struct {
	Name          string
	InstanceState string
	InstanceID    string
	PublicIP      *string
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
				Name:          getMCName(instance),
				InstanceState: *instance.State.Name,
				InstanceID:    *instance.InstanceId,
				PublicIP:      instance.PublicIpAddress,
			})
		}
	}

	return servers, nil
}

func storageKeyForName(name string) string {
	return "servers/" + name + ".tar"
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

// ReserveInstance (run) an EC2 instance
func ReserveInstance(services *Detail, name string) (string, error) {
	services.Logger.Info("reserving EC2 instance")

	reservation, err := services.EC2.RunInstances(&ec2.RunInstancesInput{
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
		ImageId:      aws.String("ami-0cb790308f7591fa6"), // Normal Amazon Linux 2.
		InstanceType: aws.String("m5.large"),              // FIXME configurable
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

	services.Logger.Info("API call complete")

	if err != nil {
		return "", err
	}

	if len(reservation.Instances) != 1 {
		return "", fmt.Errorf("runstored: reservation returned non-1 (%d) instances", len(reservation.Instances))
	}

	return *reservation.Instances[0].InstanceId, nil
}

// TerminateInstance terminates an EC2 instance.
func TerminateInstance(services *Detail, instanceID string) error {
	services.Logger.Info("terminating EC2 instance")

	_, err := services.EC2.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	})

	return err
}

// RunStored runs a Minecraft server on EC2 from a world stored on S3.
func RunStored(detail *Detail, world string) error {

	err := FindStored(detail.S3, world)
	if err != nil {
		return err
	}

	instanceID, err := ReserveInstance(detail, world)
	if err != nil {
		return err
	}

	err = SetupInstance(detail, instanceID, world)
	if err != nil {
		return err
	}

	return nil
}

// StoreRunning takes a running minecraft server and safely stops, saves, and terminates the instance.
func StoreRunning(detail *Detail, world string) error {
	server, err := FindRunning(detail.EC2, world)
	if err != nil {
		return err
	}

	if server.InstanceState == "terminated" || server.InstanceState == "shutting-down" {
		return fmt.Errorf("server not running (%s): state is %s", server.Name, server.InstanceState)
	}

	err = StopServerWrapper(detail, server.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to stop server wrapper (%s): %w", server.Name, err)
	}

	err = UploadWorld(detail, server.InstanceID, server.Name)
	if err != nil {
		return fmt.Errorf("failed to upload world (%s): %w", server.Name, err)
	}

	return TerminateInstance(detail, server.InstanceID)
}

// BootstrapInstance takes an existing EC2 instance and installs all prerequisites
// for running a minecraft server.
func BootstrapInstance(services *Detail, instanceID string) error {

	services.Logger.Info("bootstrapping instance")

	err := services.RunOn(instanceID, `
		set -x
		sudo yum update -y;
		sudo yum install -y docker;
		sudo service docker start;
		sudo usermod -a -G docker ec2-user;
	`, RunOpts{NewKeyBehaviour: SSHNewKeyAccept})

	return err
}

// DownloadWorld on remote instance.
func DownloadWorld(services *Detail, instanceID, name string) error {
	services.Logger.Info("downloading world")

	s3ObjectPath := "s3://" + s3BucketName + "/" + storageKeyForName(name)

	err := services.RunOn(instanceID, fmt.Sprintf(`
		set -x
		aws s3 cp %s server.tar
		tar xf server.tar
		rm server.tar
		sudo mv server/ /
	`, s3ObjectPath), RunOpts{})

	return err
}

// Status gets the status of an instance's server wrapper.
func Status(services *Detail, instanceID string) (serverwrapper.StatusResponse, error) {

	out, _, err := services.OutputOn(instanceID, "curl localhost:8080/status", RunOpts{})
	if err != nil {
		return serverwrapper.StatusResponse{}, fmt.Errorf("status: %w", err)
	}

	var statusResponse serverwrapper.StatusResponse
	err = json.Unmarshal(out, &statusResponse)
	return statusResponse, err
}

// UploadWorld uploads a world from an EC2 instance to S3.
func UploadWorld(services *Detail, instanceID, name string) error {
	s3ObjectPath := "s3://" + s3BucketName + "/" + storageKeyForName(name)

	// TODO: Verify the world name somehow before upload to prevent accidental overwrite?
	resp, err := Status(services, instanceID)
	if err != nil {
		return err
	}

	if resp.Status != serverwrapper.StatusStopped {
		return fmt.Errorf("server still running for world '%s', must be stopped to upload world", name)
	}

	err = services.RunOn(instanceID, fmt.Sprintf(`
		set -x
		tar cf server.tar /server
		aws s3 cp server.tar %s
		rm server.tar
	`, s3ObjectPath), RunOpts{})

	return err
}

// StartServerWrapper starts the server wrapper on the EC2 instance that the
// ssh client is connected to. Expects it isn't already running.
func StartServerWrapper(services *Detail, instanceID string) error {
	region := services.Region()
	account, err := services.Account()
	if err != nil {
		return err
	}

	err = services.RunOn(instanceID, fmt.Sprintf(`
		set -x
		# Log in to docker
		# sed hack to remove an invalid argument, god knows why it's there.
		$(aws ecr get-login --region %s | sed 's/-e none//g')
		
		docker pull %s.dkr.ecr.%s.amazonaws.com/minecloud/server-wrapper:latest

		docker run -d \
			--rm \
			-p 8080:8080 \
			-p 25565:25565 \
			--name serverwrapper \
			--volume /server:/server \
			%s.dkr.ecr.%s.amazonaws.com/minecloud/server-wrapper:latest \
			-address 0.0.0.0:8080
	`, region, account, region, account, region), RunOpts{})

	return err
}

// StopServerWrapper stops the server wrapper
func StopServerWrapper(services *Detail, instanceID string) error {
	err := services.RunOn(instanceID, "curl -X POST localhost:8080/stop", RunOpts{})
	if err != nil {
		return err
	}

	return WaitForStopped(services, instanceID)
}

// ClaimWorld for use on a server.
func ClaimWorld(detail *Detail, world string) error {
	db := dynamodb.New(detail.Session)

	_, err := db.PutItem(&dynamodb.PutItemInput{
		ConditionExpression: aws.String("attribute_not_exists(world)"),
		Item: map[string]*dynamodb.AttributeValue{
			"world": {S: aws.String(world)},
		},
		TableName: aws.String("MinecloudServers"),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				return fmt.Errorf("%w: %s", ErrWorldAlreadyClaimed, world)
			default:
			}
		}
	}

	return err
}

// UnclaimWorld for use on a server.
func UnclaimWorld(detail *Detail, world string) error {
	db := dynamodb.New(detail.Session)

	_, err := db.DeleteItem(&dynamodb.DeleteItemInput{
		ConditionExpression: aws.String("attribute_exists(world)"),
		TableName:           aws.String("MinecloudServers"),
		Key: map[string]*dynamodb.AttributeValue{
			"world": {S: aws.String(world)},
		},
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				return fmt.Errorf("%w: %s", ErrWorldNotClaimed, world)
			default:
			}
		}
	}

	return err
}

// WaitForStopped server wrapper.
func WaitForStopped(services *Detail, instanceID string) error {
	retryAttempts := 3
	var err error

	for i := 0; i < retryAttempts; i++ {
		var resp serverwrapper.StatusResponse
		resp, err = Status(services, instanceID)
		if err != nil {
			break
		}
		if resp.Status == serverwrapper.StatusStopped {
			return nil
		}
		time.Sleep(3 * time.Second)
	}

	if err == nil {
		err = errors.New("hit max retries for server wrapper stop wait")
	}

	return err
}

// WaitForSSH waits for an instance to have SSH available.
func WaitForSSH(services *Detail, instanceID string, acceptNewKey bool) error {
	services.Logger.Info("waiting for instance to be running")

	err := services.EC2.WaitUntilInstanceRunning(descInput(instanceID))
	if err != nil {
		return err
	}

	retryAttempts := 3

	for i := 0; i < retryAttempts; i++ {
		services.Logger.Info("Attempting SSH connection...")
		_, _, err = services.OutputOn(instanceID, "ls", RunOpts{NewKeyBehaviour: SSHNewKeyAccept})
		if err == nil {
			services.Logger.Info("SSH established")
			return nil
		}

		services.Logger.Info("SSH error:", err)
		time.Sleep(5 * time.Second)
	}

	return err
}

// SetupInstance sets up an existing EC2 instance into a Minecraft server.
func SetupInstance(services *Detail, instanceID, name string) error {

	err := WaitForSSH(services, instanceID, true)
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

// IsActiveInstanceState returns true if a state represents a running, not-shutting-down instance.
func IsActiveInstanceState(state string) bool {
	return state == "running"
}

// FindRunning returns the server if it exists. Error will be ErrServerNotFound if
// not found, and a different error otherwise.
func FindRunning(svc *ec2.EC2, name string) (MCServer, error) {
	servers, err := GetRunning(svc)
	if err != nil {
		return MCServer{}, err
	}

	for _, server := range servers {
		if server.Name == name && IsActiveInstanceState(server.InstanceState) {
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
