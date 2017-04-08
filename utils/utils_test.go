package utils

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
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

	err := HandleRequest(&req)

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

	err := HandleRequest(&req)

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

	err := HandleRequest(&req)

	if err != nil {
		t.Errorf("handleRequest returned nil when req.Send() returned RequestLimitExceeded then nil. %v", err)
	}
}

func TestHandleRequestThrottling(t *testing.T) {
	req := MockRequest{}
	mockErr := errors.New("test error")
	awsErr := awserr.New("Throttling", "message", mockErr)
	requestFailure := awserr.NewRequestFailure(awsErr, 1, "1")
	count := 0
	req.SendFunc = func() error {
		if count == 0 {
			count++
			return requestFailure
		}
		return nil
	}

	err := HandleRequest(&req)

	if err != nil {
		t.Errorf("handleRequest returned nil when req.Send() returned Throttling then nil. %v", err)
	}
}
