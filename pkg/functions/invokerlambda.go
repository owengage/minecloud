package functions

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
)

type LambdaInvoker struct {
	LS *lambda.Lambda
}

func (invoker *LambdaInvoker) Invoke(name string, payload []byte) error {
	_, err := invoker.LS.Invoke(&lambda.InvokeInput{
		FunctionName:   &name,
		InvocationType: aws.String(lambda.InvocationTypeEvent),
		Payload:        payload,
	})
	return err
}
