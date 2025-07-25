// Code generated by go-swagger; DO NOT EDIT.

package run_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
)

// RunServiceRetryRunReader is a Reader for the RunServiceRetryRun structure.
type RunServiceRetryRunReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *RunServiceRetryRunReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewRunServiceRetryRunOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewRunServiceRetryRunDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewRunServiceRetryRunOK creates a RunServiceRetryRunOK with default headers values
func NewRunServiceRetryRunOK() *RunServiceRetryRunOK {
	return &RunServiceRetryRunOK{}
}

/*
RunServiceRetryRunOK describes a response with status code 200, with default header values.

A successful response.
*/
type RunServiceRetryRunOK struct {
	Payload interface{}
}

// IsSuccess returns true when this run service retry run o k response has a 2xx status code
func (o *RunServiceRetryRunOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this run service retry run o k response has a 3xx status code
func (o *RunServiceRetryRunOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this run service retry run o k response has a 4xx status code
func (o *RunServiceRetryRunOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this run service retry run o k response has a 5xx status code
func (o *RunServiceRetryRunOK) IsServerError() bool {
	return false
}

// IsCode returns true when this run service retry run o k response a status code equal to that given
func (o *RunServiceRetryRunOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the run service retry run o k response
func (o *RunServiceRetryRunOK) Code() int {
	return 200
}

func (o *RunServiceRetryRunOK) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /apis/v2beta1/runs/{run_id}:retry][%d] runServiceRetryRunOK %s", 200, payload)
}

func (o *RunServiceRetryRunOK) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /apis/v2beta1/runs/{run_id}:retry][%d] runServiceRetryRunOK %s", 200, payload)
}

func (o *RunServiceRetryRunOK) GetPayload() interface{} {
	return o.Payload
}

func (o *RunServiceRetryRunOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewRunServiceRetryRunDefault creates a RunServiceRetryRunDefault with default headers values
func NewRunServiceRetryRunDefault(code int) *RunServiceRetryRunDefault {
	return &RunServiceRetryRunDefault{
		_statusCode: code,
	}
}

/*
RunServiceRetryRunDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type RunServiceRetryRunDefault struct {
	_statusCode int

	Payload *run_model.GooglerpcStatus
}

// IsSuccess returns true when this run service retry run default response has a 2xx status code
func (o *RunServiceRetryRunDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this run service retry run default response has a 3xx status code
func (o *RunServiceRetryRunDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this run service retry run default response has a 4xx status code
func (o *RunServiceRetryRunDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this run service retry run default response has a 5xx status code
func (o *RunServiceRetryRunDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this run service retry run default response a status code equal to that given
func (o *RunServiceRetryRunDefault) IsCode(code int) bool {
	return o._statusCode == code
}

// Code gets the status code for the run service retry run default response
func (o *RunServiceRetryRunDefault) Code() int {
	return o._statusCode
}

func (o *RunServiceRetryRunDefault) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /apis/v2beta1/runs/{run_id}:retry][%d] RunService_RetryRun default %s", o._statusCode, payload)
}

func (o *RunServiceRetryRunDefault) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /apis/v2beta1/runs/{run_id}:retry][%d] RunService_RetryRun default %s", o._statusCode, payload)
}

func (o *RunServiceRetryRunDefault) GetPayload() *run_model.GooglerpcStatus {
	return o.Payload
}

func (o *RunServiceRetryRunDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.GooglerpcStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
