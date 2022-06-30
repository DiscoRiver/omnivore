package ec2

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// GetClient returns a new ec2 client from the given config
func GetClient(c aws.Config) *ec2.Client {
	return ec2.NewFromConfig(c)
}
