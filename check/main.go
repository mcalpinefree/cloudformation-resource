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
	result, marshalErr := json.Marshal(getVersions(input, svc, &utils.AwsRequestHandler{}))
	utils.Logln("Result is: ", result)
	if marshalErr != nil {
		utils.Logln("Error occured marshalling output: ", marshalErr)
		os.Exit(1)
	}
	fmt.Printf("%s", result)
}

func getVersions(input utils.Input, svc utils.AwsCloudformationSvc, requestHandler utils.RequestHandler) []Version {
	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(input.Source.Name),
	}
	req, resp := svc.DescribeStacksRequest(params)

	err := requestHandler.HandleRequest(req)

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
		result = append(result, Version{LastUpdatedTime: "nil"})
		return result
	}

	// There is a new version available
	result := []Version{}
	result = append(result, Version{LastUpdatedTime: input.Version.LastUpdatedTime})
	result = append(result, Version{LastUpdatedTime: newVersion})
	return result
}
