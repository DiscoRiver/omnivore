// Package test contains relevant test parameters.
package test

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/pkg/aws/ec2"
	"github.com/discoriver/omnivore/pkg/aws/filters"
	"golang.org/x/crypto/ssh"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/discoriver/omnivore/internal/log"
)

var (
	Job = &massh.Job{
		Command: "echo \"Hello, World\"",
	}

	SSHConfig = &ssh.ClientConfig{
		User:            "ubuntu",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(5) * time.Second,
	}

	Config = &massh.Config{
		Hosts:      map[string]struct{}{},
		SSHConfig:  SSHConfig,
		Job:        Job,
		WorkerPool: 10,
	}

	AWSInstances           []types.Instance
	AWSWaitForStartSeconds = 20
)

func InitTestLogger() {
	log.OmniLog = &log.OmniLogger{FileOutput: os.Stdout}
	log.OmniLog.Init()
}

func InitAWSHosts(t *testing.T) {
	SetAWSHosts(t)
}

func StopAWSTestHosts(t *testing.T) {
	// Environment variables for IAM user key should be gathered. They should exist in the Github workspace environment variables if running there.
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Logf("Failed to load default AWS config: %s", err)
		t.FailNow()
	}

	client := ec2.GetClient(cfg)

	err = ec2.StopAllInstances(context.TODO(), client, AWSInstances)
	if err != nil {
		t.Logf("Error stopping instances post-test, does not affect test outcome, but may be worth checking aws console manually: %s", err)
	}
}

func SetAWSHosts(t *testing.T) {
	Config.SetHosts(GetAWSTestHosts(t))
}

func GetAWSTestHosts(t *testing.T) []string {
	// Environment variables for IAM user key should be gathered. They should exist in the Github workspace environment variables if running there.
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		t.Logf("Failed to load default AWS config: %s", err)
		t.FailNow()
	}

	client := ec2.GetClient(cfg)

	f := map[string][]string{}
	// Filter for omnivore EC2 group.
	f["network-interface.group-name"] = []string{"omnivore"}

	instancesDescription, err := ec2.GetInstancesWithFilters(context.TODO(), client, filters.GenerateFilterSlice(f))
	if err != nil {
		t.Logf("Couldn't get AWS instances: %s", err)
		t.FailNow()
	}

	AWSInstances = ec2.GetInstancesFromReservasions(instancesDescription.Reservations)
	if ec2.AnyInstanceIsNotRunning(AWSInstances) {
		err := ec2.StartAllInstancesWait(context.TODO(), client, AWSInstances, AWSWaitForStartSeconds)
		if err != nil {
			t.Logf("Didn't start all EC2 instances for test: %s", err)
			t.FailNow()
		}
	}

	return ec2.GetInstancesPublicDnsName(AWSInstances)
}

func ReadStreamWithTimeout(res *massh.Result, timeout time.Duration, wg *sync.WaitGroup, t *testing.T) {
	timer := time.NewTimer(timeout)
	defer func() {
		timer.Stop()
		wg.Done()
	}()

	for {
		select {
		case d := <-res.StdOutStream:
			t.Logf("%s: %s\n", res.Host, d)
			timer.Reset(timeout)
		case e := <-res.StdErrStream:
			t.Logf("%s: %s\n", res.Host, e)
			timer.Reset(timeout)
		case <-res.DoneChannel:
			// Confirm that the host has exited.
			t.Logf("Host %s finished.\n", res.Host)
			timer.Reset(timeout)
			return
		case <-timer.C:
			t.Logf("Activity timeout: %s\n", res.Host)
			return
		}
	}
}
