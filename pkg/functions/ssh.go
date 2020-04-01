package functions

import (
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/owengage/minecloud/pkg/awsdetail"
)

// GetSSHKey from S3.
func GetSSHKey(awsSession *session.Session) []byte {
	sss := s3.New(awsSession)

	req, err := sss.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(awsdetail.SecretsBucketName),
		Key:    aws.String("MinecraftServerKeyPair.pem"),
	})
	if err != nil {
		panic("could not request SSH key from S3")
	}

	keyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic("could not read SSH key")
	}

	return keyBytes
}
