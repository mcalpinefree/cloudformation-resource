package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

type AwsRequest interface {
	Send() error
}

type RequestHandler interface {
	HandleRequest(req AwsRequest) error
}

type AwsRequestHandler struct {
}

func (r *AwsRequestHandler) HandleRequest(req AwsRequest) error {
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
	Params struct {
		Template     string   `json:"template"`
		Parameters   string   `json:"parameters"`
		Tags         string   `json:"tags"`
		Capabilities []string `json:"capabilities"`
		Delete       bool     `json:"delete"`
		Wait         bool     `json:"wait"`
	} `json:"params"`
}

func GetInput() Input {
	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	input := Input{}
	err = json.Unmarshal(bytes, &input)
	if err != nil {
		panic(err)
	}
	return input
}

type AwsCloudformationSvc interface {
	DescribeStacksRequest(input *cloudformation.DescribeStacksInput) (req *request.Request, output *cloudformation.DescribeStacksOutput)
	CreateStackRequest(input *cloudformation.CreateStackInput) (req *request.Request, output *cloudformation.CreateStackOutput)
	UpdateStackRequest(input *cloudformation.UpdateStackInput) (req *request.Request, output *cloudformation.UpdateStackOutput)
	DescribeStackEventsRequest(input *cloudformation.DescribeStackEventsInput) (req *request.Request, output *cloudformation.DescribeStackEventsOutput)
	DeleteStackRequest(input *cloudformation.DeleteStackInput) (req *request.Request, output *cloudformation.DeleteStackOutput)
}

func GetCloudformationService(input Input) AwsCloudformationSvc {
	creds := credentials.NewStaticCredentials(input.Source.AwsAccessKeyId, input.Source.AwsSecretAccessKey, "")
	awsConfig := aws.NewConfig().WithCredentials(creds).WithRegion(input.Source.Region)
	sess := session.Must(session.NewSession(awsConfig))
	svc := cloudformation.New(sess)
	return svc
}

func GoToBuildDirectory() {
	files, err := ioutil.ReadDir("/tmp/build")
	if err != nil {
		panic(err)
	}

	if len(files) != 1 {
		fmt.Printf("Expected only 1 file in /tmp/build but found %d: %v\n", len(files), files)
		os.Exit(1)
	}

	os.Chdir("/tmp/build/" + files[0].Name())
}
