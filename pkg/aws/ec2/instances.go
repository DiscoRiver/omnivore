// Package ec2 contains methods for interacting and obtaining AWS instances
package ec2

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"time"
)

// EC2DescribeInstancesAPI defines the interface for the DescribeInstances function.
// We use this interface to test the function using a mocked service.
type EC2DescribeInstancesAPI interface {
	DescribeInstances(ctx context.Context,
		params *ec2.DescribeInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

type EC2StartInstancesAPI interface {
	StartInstances(ctx context.Context,
		params *ec2.StartInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error)
}

type EC2StopInstancesAPI interface {
	StopInstances(ctx context.Context,
		params *ec2.StopInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error)
}

// GetInstances retrieves information about your Amazon Elastic Compute Cloud (Amazon EC2) instances.
// Inputs:
//     c is the context of the method call, which includes the AWS Region.
//     api is the interface that defines the method call.
//     input defines the input arguments to the service call.
// Output:
//     If success, a DescribeInstancesOutput object containing the result of the service call and nil.
//     Otherwise, nil and an error from the call to DescribeInstances.
func getInstances(c context.Context, api EC2DescribeInstancesAPI, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	return api.DescribeInstances(c, input)
}

func startInstances(c context.Context, api EC2StartInstancesAPI, input *ec2.StartInstancesInput) (*ec2.StartInstancesOutput, error) {
	return api.StartInstances(c, input)
}

func stopInstances(c context.Context, api EC2StopInstancesAPI, input *ec2.StopInstancesInput) (*ec2.StopInstancesOutput, error) {
	return api.StopInstances(c, input)
}

// GetInstancesWithFilters returns instances matching filters.
func GetInstancesWithFilters(c context.Context, client EC2DescribeInstancesAPI, f []types.Filter) (*ec2.DescribeInstancesOutput, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: f,
	}

	results, err := getInstances(c, client, input)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func GetInstancesFromReservasions(reservations []types.Reservation) (instances []types.Instance) {
	for _, r := range reservations {
		instances = append(instances, r.Instances...)
	}
	return
}

func AnyInstanceIsNotRunning(instances []types.Instance) bool {
	for i := range instances {
		if *instances[i].State.Code != 16 {
			fmt.Println(*instances[i].PublicDnsName, "", *instances[i].State.Code)
			return true
		}
	}
	return false
}

func AnyInstanceIsNotStoppedOrStopping(instances []types.Instance) bool {
	for i := range instances {
		if *instances[i].State.Code != 80 && *instances[i].State.Code != 64 && *instances[i].State.Code != 32 {
			return true
		}
	}
	return false
}

func StartAllInstancesWait(c context.Context, client EC2StartInstancesAPI, instances []types.Instance, waitForSeconds int) error {
	var ids []string
	for i := range instances {
		ids = append(ids, *instances[i].InstanceId)
	}

	input := &ec2.StartInstancesInput{
		InstanceIds: ids,
	}

	_, err := startInstances(c, client, input)
	if err != nil {
		return err
	}

	time.Sleep(time.Duration(waitForSeconds) * time.Second)
	return nil
}

// StopAllInstancesWait will attempt to stop all instances,
func StopAllInstances(c context.Context, client EC2StopInstancesAPI, instances []types.Instance) error {
	var ids []string
	for i := range instances {
		ids = append(ids, *instances[i].InstanceId)
	}

	input := &ec2.StopInstancesInput{
		InstanceIds: ids,
	}

	_, err := stopInstances(c, client, input)
	if err != nil {
		return err
	}
	return nil
}

func GetInstancesPublicDnsName(instances []types.Instance) (publicDnsNames []string) {
	for i := range instances {
		publicDnsNames = append(publicDnsNames, *instances[i].PublicDnsName)
	}
	return
}
