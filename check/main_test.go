package main

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/ci-pipeline/cloudformation-resource/utils"
)

type MockRequest struct {
	SendFunc (func() error)
}

func (r *MockRequest) Send() error {
	return r.SendFunc()
}

func TestHandleRequestNoError(t *testing.T) {
	req := MockRequest{}
	req.SendFunc = func() error {
		return nil
	}

	requestHandler := &utils.AwsRequestHandler{}

	err := requestHandler.HandleRequest(&req)

	if err != nil {
		t.Error("handleRequest returned an error when req.Send() returned nil")
	}

}

func TestHandleRequestRequestFailure(t *testing.T) {
	req := MockRequest{}
	mockErr := errors.New("test error")
	awsErr := awserr.New("1", "test error", mockErr)
	requestFailure := awserr.NewRequestFailure(awsErr, 1, "1")
	req.SendFunc = func() error {
		return requestFailure
	}

	requestHandler := &utils.AwsRequestHandler{}

	err := requestHandler.HandleRequest(&req)

	if err == nil {
		t.Error("handleRequest returned nil when req.Send() returned an error")
	}

	if err != requestFailure {
		t.Error("handleRequest(&req) != requestFailure")
	}
}

func TestHandleRequestRequestLimitExceeded(t *testing.T) {
	req := MockRequest{}
	mockErr := errors.New("test error")
	awsErr := awserr.New("RequestLimitExceeded", "message", mockErr)
	requestFailure := awserr.NewRequestFailure(awsErr, 1, "1")
	count := 0
	req.SendFunc = func() error {
		if count == 0 {
			count++
			return requestFailure
		}
		return nil
	}

	requestHandler := &utils.AwsRequestHandler{}

	err := requestHandler.HandleRequest(&req)

	if err != nil {
		t.Errorf("handleRequest returned nil when req.Send() returned RequestLimitExceeded then nil. %v", err)
	}
}

type MockCloudformationSvc struct {
	Request  *request.Request
	Response *cloudformation.DescribeStacksOutput
}

func (s *MockCloudformationSvc) DescribeStacksRequest(input *cloudformation.DescribeStacksInput) (req *request.Request, output *cloudformation.DescribeStacksOutput) {
	return s.Request, s.Response
}

type MockRequestHandler struct {
	Response error
}

func (r *MockRequestHandler) HandleRequest(req utils.AwsRequest) error {
	return r.Response
}

func TestGetVersionsNoStack(t *testing.T) {
	input := utils.Input{}

	svc := &MockCloudformationSvc{}
	requestHandler := &MockRequestHandler{Response: errors.New("Stack does not exist")}

	versions := getVersions(input, svc, requestHandler)

	if len(versions) > 0 {
		t.Error("Versions was %v but should have been an empty slice")
	}
}

func TestGetVersionsNoPrevious(t *testing.T) {
	input := utils.Input{}

	stacks := []*cloudformation.Stack{&cloudformation.Stack{}}
	svc := &MockCloudformationSvc{Response: &cloudformation.DescribeStacksOutput{Stacks: stacks}}
	requestHandler := &MockRequestHandler{}

	versions := getVersions(input, svc, requestHandler)

	if len(versions) != 1 {
		t.Errorf("Versions was not length 1 but length %d: %v", len(versions), versions)
	}

	if versions[0] != "nil" {
		t.Errorf("Expected versions[0] == \"nil\", but was %v", versions[0])
	}
}

func TestGetVersionsWithPreviousTheSame(t *testing.T) {
	input := utils.Input{}

	now := time.Now()
	input.Version.LastUpdatedTime = now.String()
	stacks := []*cloudformation.Stack{&cloudformation.Stack{LastUpdatedTime: &now}}
	svc := &MockCloudformationSvc{Response: &cloudformation.DescribeStacksOutput{Stacks: stacks}}
	requestHandler := &MockRequestHandler{}

	versions := getVersions(input, svc, requestHandler)

	if len(versions) != 1 {
		t.Errorf("Versions was not length 1 but length %d: %v", len(versions), versions)
	}

	if versions[0] != now.String() {
		t.Errorf("Expected versions[0] == \"\", but was %v", versions[0])
	}
}

func TestGetVersionsWithPreviousDifferent(t *testing.T) {
	input := utils.Input{}
	input.Version.LastUpdatedTime = time.Now().String()
	time.Sleep(5 * time.Millisecond)

	now := time.Now()
	stacks := []*cloudformation.Stack{&cloudformation.Stack{LastUpdatedTime: &now}}
	svc := &MockCloudformationSvc{Response: &cloudformation.DescribeStacksOutput{Stacks: stacks}}
	requestHandler := &MockRequestHandler{}

	versions := getVersions(input, svc, requestHandler)

	if len(versions) != 2 {
		t.Errorf("Versions was not length 2 but length %d: %v", len(versions), versions)
	}

	if versions[0] != input.Version.LastUpdatedTime {
		t.Errorf("Expected versions[0] == \"\", but was %v", versions[0])
	}

	if versions[1] != now.String() {
		t.Errorf("Expected versions[1] == \"%s\", but was %v", now.String(), versions[1])
	}
}
