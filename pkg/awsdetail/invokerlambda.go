package awsdetail

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
)

// LambdaInvoker invokes lambda functions on AWS.
type LambdaInvoker struct {
	LS *lambda.Lambda
}

// Invoke lambda function.
func (invoker *LambdaInvoker) Invoke(name string, payload []byte) error {
	_, err := invoker.LS.Invoke(&lambda.InvokeInput{
		FunctionName:   &name,
		InvocationType: aws.String(lambda.InvocationTypeEvent),
		Payload:        payload,
	})
	return err
}
