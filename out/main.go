package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/ci-pipeline/cloudformation-resource/utils"
)

func main() {
	utils.Logln("Change to build directory")
	utils.GoToBuildDirectory()
	cwd, _ := os.Getwd()
	utils.Logln(cwd)
	input := utils.GetInput()
	svc := utils.GetCloudformationService(input)
	metadata, success := out(input, svc, &utils.AwsRequestHandler{})
	fmt.Printf("%s", metadata)

	if !success {
		os.Exit(1)
	}
}

type Parameter struct {
	ParameterValue   string
	ParameterKey     string
	UsePreviousValue bool
}

type Tag struct {
	TagKey   string
	TagValue string
}

func stackExists(reqHandler utils.RequestHandler, svc utils.AwsCloudformationSvc, stackName string) bool {
	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	req, resp := svc.DescribeStacksRequest(params)
	err := reqHandler.HandleRequest(req)
	if err != nil {
		return false
	}
	utils.Logln(resp)

	return resp.Stacks[0] != nil
}

func waitForStack(reqHandler utils.RequestHandler, svc utils.AwsCloudformationSvc, input utils.Input) (success bool, arn, timestamp string) {
	success = true
	params := &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(input.Source.Name),
	}

	pos := 0
	var stackEvents []*cloudformation.StackEvent
	for {
		req, resp := svc.DescribeStackEventsRequest(params)
		err := reqHandler.HandleRequest(req)
		if err != nil {
			if input.Params.Delete {
				success = true
				utils.Logln("Stack deleted")
				return
			}
			utils.Logf("An error occured: %v\n", err)
			return
		}
		stackEvents = resp.StackEvents
		for j := len(stackEvents) - 1 - pos; j > -1; j-- {
			utils.Logln(stackEvents[j])
		}
		pos = len(stackEvents)

		arn = *stackEvents[0].StackId
		timestamp = stackEvents[0].Timestamp.String()

		if *stackEvents[0].ResourceType == "AWS::CloudFormation::Stack" {
			if *stackEvents[0].ResourceStatus == "CREATE_COMPLETE" {
				return
			} else if *stackEvents[0].ResourceStatus == "UPDATE_COMPLETE" {
				return
			} else if *stackEvents[0].ResourceStatus == "ROLLBACK_COMPLETE" {
				success = false
				return
			} else if *stackEvents[0].ResourceStatus == "UPDATE_ROLLBACK_COMPLETE" {
				success = false
				return
			} else if strings.HasSuffix(*stackEvents[0].ResourceStatus, "_FAILED") {
				success = false
				return
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func out(input utils.Input, svc utils.AwsCloudformationSvc, reqHandler utils.RequestHandler) (metadata string, success bool) {

	var capabilities []*string
	capabilities = nil
	if input.Params.Capabilities != nil {
		for _, c := range input.Params.Capabilities {
			capabilities = append(capabilities, aws.String(c))
		}
	}

	var templateBody string
	if input.Params.Template != "" {
		bytes, err := ioutil.ReadFile(input.Params.Template)
		if err != nil {
			panic(err)
		}
		templateBody = string(bytes)
	}

	parameters := []Parameter{}
	var cloudformationParams []*cloudformation.Parameter
	if input.Params.Parameters != "" {
		bytes, err := ioutil.ReadFile(input.Params.Parameters)
		err = json.Unmarshal(bytes, &parameters)
		if err != nil {
			panic(err)
		}
	}
	for _, p := range parameters {
		param := cloudformation.Parameter{
			ParameterKey:     aws.String(p.ParameterKey),
			ParameterValue:   aws.String(p.ParameterValue),
			UsePreviousValue: aws.Bool(p.UsePreviousValue),
		}
		cloudformationParams = append(cloudformationParams, &param)
	}

	tags := []Tag{}
	var cloudformationTags []*cloudformation.Tag
	if input.Params.Tags != "" {
		bytes, err := ioutil.ReadFile(input.Params.Tags)
		err = json.Unmarshal(bytes, &tags)
		if err != nil {
			panic(err)
		}
	}
	for _, t := range tags {
		tag := cloudformation.Tag{
			Key:   aws.String(t.TagKey),
			Value: aws.String(t.TagValue),
		}
		cloudformationTags = append(cloudformationTags, &tag)
	}

	if input.Params.Delete == true {
		utils.Logln("Deleting stack")
		params := &cloudformation.DeleteStackInput{
			StackName: aws.String(input.Source.Name),
		}
		req, resp := svc.DeleteStackRequest(params)
		err := reqHandler.HandleRequest(req)
		if err != nil {
			utils.Logln(err.Error())
		}
		utils.Logln(resp)
	} else if !stackExists(reqHandler, svc, input.Source.Name) {
		utils.Logln("Creating stack")
		params := &cloudformation.CreateStackInput{
			StackName:    aws.String(input.Source.Name),
			Capabilities: capabilities,
			Parameters:   cloudformationParams,
			Tags:         cloudformationTags,
			TemplateBody: aws.String(templateBody),
		}
		req, resp := svc.CreateStackRequest(params)
		err := reqHandler.HandleRequest(req)
		if err != nil {
			utils.Logln(err.Error())
		}
		utils.Logln(resp)
	} else {
		utils.Logln("Updating stack")
		params := &cloudformation.UpdateStackInput{
			StackName:    aws.String(input.Source.Name),
			Capabilities: capabilities,
			Parameters:   cloudformationParams,
			Tags:         cloudformationTags,
			TemplateBody: aws.String(templateBody),
		}
		req, resp := svc.UpdateStackRequest(params)
		err := reqHandler.HandleRequest(req)
		if err != nil {
			utils.Logln(err.Error())
		}
		utils.Logln(resp)
	}

	success, arn, timestamp := waitForStack(reqHandler, svc, input)
	result := make(map[string]string)
	result["arn"] = arn
	result["timestamp"] = timestamp
	bytes, _ := json.Marshal(result)
	return string(bytes), success
}
