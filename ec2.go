package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Ec2Client struct {
	ClientConnection *ec2.EC2
}

func Ec2Init(region string) *Ec2Client {
	var ret Ec2Client
	ret.ClientConnection = ec2.New(session.New(), &aws.Config{Region: aws.String(region)})
	return &ret
}

func (c *Ec2Client) DescribeInstance(id string) (*ec2.Instance, error) {
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-id"),
				Values: []*string{
					aws.String(id),
				},
			},
		},
	}

	resp, err := c.ClientConnection.DescribeInstances(params)
	if err != nil {
		fmt.Printf("Error getting instance: %s\n", err)
		return nil, err
	}
	return resp.Reservations[0].Instances[0], nil
}

func (c Ec2Client) DescribeInstances(output chan *ec2.Instance) error {
	params := &ec2.DescribeInstancesInput{}

	resp, err := c.ClientConnection.DescribeInstances(params)
	if err != nil {
		fmt.Printf("Error getting all instances %s\n", err)
		return err
	}

	for _, r := range resp.Reservations {
		for _, i := range r.Instances {
			output <- i
		}
	}

	return nil
}
