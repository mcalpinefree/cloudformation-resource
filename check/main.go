package main

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"

	"encoding/json"
	"fmt"

	"github.com/ci-pipeline/cloudformation-resource/utils"
)

type Version struct {
	LastUpdatedTime string
}

func main() {
	input := utils.GetInput()
	svc := utils.GetCloudformationService(input)
	resp, err := describeStack(input, svc)
	result, marshalErr := json.Marshal(getVersions(input, resp, err))
	utils.Logln("Result is: ", result)
	if marshalErr != nil {
		utils.Logln("Error occured marshalling output: ", marshalErr)
		os.Exit(1)
	}
	fmt.Printf("%s\n", result)
}

func getVersions(input utils.Input, resp *cloudformation.DescribeStacksOutput, err error) []Version {
	// Stack does not exists, return empty list
	if err != nil {
		return []Version{}
	}

	lastUpdatedTime := resp.Stacks[0].LastUpdatedTime

	// First version of stack
	if lastUpdatedTime == nil {
		result := []Version{}
		result = append(result, Version{LastUpdatedTime: "nil"})
		return result
	}

	newVersion := lastUpdatedTime.String()

	// Same as current version
	if input.Version.LastUpdatedTime == newVersion {
		result := []Version{}
		result = append(result, Version{LastUpdatedTime: newVersion})
		return result
	}

	// There is a new version available
	result := []Version{}
	result = append(result, Version{LastUpdatedTime: input.Version.LastUpdatedTime})
	result = append(result, Version{LastUpdatedTime: newVersion})
	return result
}

func describeStack(input utils.Input, svc *cloudformation.CloudFormation) (*cloudformation.DescribeStacksOutput, error) {
	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(input.Source.Name),
	}
	req, resp := svc.DescribeStacksRequest(params)

	err := utils.HandleRequest(req)

	return resp, err
}
