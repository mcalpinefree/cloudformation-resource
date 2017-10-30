package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/concourse/atc"
)

type VersionResult struct {
	Version  atc.Version         `json:"version,omitempty"`
	Metadata []atc.MetadataField `json:"metadata,omitempty"`
}

type Input struct {
	Source struct {
		Name               string `json:"name"`
		AwsAccessKeyId     string `json:"aws_access_key_id,omitempty"`
		AwsSecretAccessKey string `json:"aws_secret_access_key,omitempty"`
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

func GetCloudformationService(input Input) *cloudformation.CloudFormation {
	creds := credentials.NewStaticCredentials(input.Source.AwsAccessKeyId, input.Source.AwsSecretAccessKey, "")
	var awsConfig *aws.Config
	if input.Source.AwsAccessKeyId != "" {
		awsConfig = aws.NewConfig().WithCredentials(creds).WithRegion(input.Source.Region).WithMaxRetries(50)
	} else {
		awsConfig = aws.NewConfig().WithRegion(input.Source.Region).WithMaxRetries(50)
	}
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
		Logf("Expected only 1 file in /tmp/build but found %d: %v\n", len(files), files)
		os.Exit(1)
	}

	os.Chdir("/tmp/build/" + files[0].Name())
}

func Logln(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
}

func Logf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a)
}
