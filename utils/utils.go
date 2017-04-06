package utils

import (
	"encoding/json"
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
}

func GetCloudformationService(input Input) AwsCloudformationSvc {
	creds := credentials.NewStaticCredentials(input.Source.AwsAccessKeyId, input.Source.AwsSecretAccessKey, "")
	awsConfig := aws.NewConfig().WithCredentials(creds).WithRegion(input.Source.Region)
	sess := session.Must(session.NewSession(awsConfig))
	svc := cloudformation.New(sess)
	return svc
}
