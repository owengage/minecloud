package minecloud

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
)

// NewAWS makes a new AWS helper object
func NewAWS(sess *session.Session) *AWS {
	return &AWS{
		Session: sess,
		EC2:     ec2.New(sess),
		S3:      s3.New(sess),
	}
}

// AWS contains useful bits for working with AWS.
type AWS struct {
	Session *session.Session
	EC2     *ec2.EC2
	S3      *s3.S3
	account *string
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
