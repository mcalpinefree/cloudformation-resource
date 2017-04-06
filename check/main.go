package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type Input struct {
	Source struct {
		Name               string `json:"name"`
		AwsAccessKeyId     string `json:"aws_access_key_id"`
		AwsSecretAccessKey string `json:"aws_secret_access_key"`
		Region             string `json:"region"`
	} `json:"source"`
	Version struct {
		LastUpdatedTime string `json:"LastUpdatedTime"`
	} `json:"version"`
}

func main() {
	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	input := Input{}
	err = json.Unmarshal(bytes, &input)
	if err != nil {
		panic(err)
	}

	creds := credentials.NewStaticCredentials(input.Source.AwsAccessKeyId, input.Source.AwsSecretAccessKey, "")
	awsConfig := aws.NewConfig().WithCredentials(creds).WithRegion(input.Source.Region)
	sess := session.Must(session.NewSession(awsConfig))
	svc := cloudformation.New(sess)

	result, marshalErr := json.Marshal(getVersions(input, svc, &AwsRequestHandler{}))
	if marshalErr != nil {
		panic(marshalErr)
	}
	fmt.Printf("%s", result)
}

type AwsCloudformationSvc interface {
	DescribeStacksRequest(input *cloudformation.DescribeStacksInput) (req *request.Request, output *cloudformation.DescribeStacksOutput)
}

func getVersions(input Input, svc AwsCloudformationSvc, requestHandler RequestHandler) []string {
	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(input.Source.Name),
	}
	req, resp := svc.DescribeStacksRequest(params)

	err := requestHandler.handleRequest(req)

	// Stack does not exists, return empty list
	if err != nil {
		return []string{}
	}

	lastUpdatedTime := resp.Stacks[0].LastUpdatedTime

	// First version of stack
	if lastUpdatedTime == nil {
		return []string{"nil"}
	}

	newVersion := lastUpdatedTime.String()

	// Same as current version
	if input.Version.LastUpdatedTime == newVersion {
		return []string{input.Version.LastUpdatedTime}
	}

	// There is a new version available
	return []string{input.Version.LastUpdatedTime, newVersion}
}

type AwsRequest interface {
	Send() error
}

type RequestHandler interface {
	handleRequest(req AwsRequest) error
}

type AwsRequestHandler struct {
}

func (r *AwsRequestHandler) handleRequest(req AwsRequest) error {
	s := 1
	var err error
	for err = req.Send(); err != nil; err = req.Send() {
		if reqerr, ok := err.(awserr.RequestFailure); ok {
			if reqerr.Code() == "RequestLimitExceeded" {
				time.Sleep(time.Duration(s) * time.Second)
				s = s * 2
				continue
			}
		}
		return err
	}
	return nil
}
