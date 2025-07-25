// Code generated by go-swagger; DO NOT EDIT.

package experiment_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
)

// ExperimentServiceUnarchiveExperimentReader is a Reader for the ExperimentServiceUnarchiveExperiment structure.
type ExperimentServiceUnarchiveExperimentReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ExperimentServiceUnarchiveExperimentReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewExperimentServiceUnarchiveExperimentOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewExperimentServiceUnarchiveExperimentDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewExperimentServiceUnarchiveExperimentOK creates a ExperimentServiceUnarchiveExperimentOK with default headers values
func NewExperimentServiceUnarchiveExperimentOK() *ExperimentServiceUnarchiveExperimentOK {
	return &ExperimentServiceUnarchiveExperimentOK{}
}

/*
ExperimentServiceUnarchiveExperimentOK describes a response with status code 200, with default header values.

A successful response.
*/
type ExperimentServiceUnarchiveExperimentOK struct {
	Payload interface{}
}

// IsSuccess returns true when this experiment service unarchive experiment o k response has a 2xx status code
func (o *ExperimentServiceUnarchiveExperimentOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this experiment service unarchive experiment o k response has a 3xx status code
func (o *ExperimentServiceUnarchiveExperimentOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this experiment service unarchive experiment o k response has a 4xx status code
func (o *ExperimentServiceUnarchiveExperimentOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this experiment service unarchive experiment o k response has a 5xx status code
func (o *ExperimentServiceUnarchiveExperimentOK) IsServerError() bool {
	return false
}

// IsCode returns true when this experiment service unarchive experiment o k response a status code equal to that given
func (o *ExperimentServiceUnarchiveExperimentOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the experiment service unarchive experiment o k response
func (o *ExperimentServiceUnarchiveExperimentOK) Code() int {
	return 200
}

func (o *ExperimentServiceUnarchiveExperimentOK) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /apis/v2beta1/experiments/{experiment_id}:unarchive][%d] experimentServiceUnarchiveExperimentOK %s", 200, payload)
}

func (o *ExperimentServiceUnarchiveExperimentOK) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /apis/v2beta1/experiments/{experiment_id}:unarchive][%d] experimentServiceUnarchiveExperimentOK %s", 200, payload)
}

func (o *ExperimentServiceUnarchiveExperimentOK) GetPayload() interface{} {
	return o.Payload
}

func (o *ExperimentServiceUnarchiveExperimentOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewExperimentServiceUnarchiveExperimentDefault creates a ExperimentServiceUnarchiveExperimentDefault with default headers values
func NewExperimentServiceUnarchiveExperimentDefault(code int) *ExperimentServiceUnarchiveExperimentDefault {
	return &ExperimentServiceUnarchiveExperimentDefault{
		_statusCode: code,
	}
}

/*
ExperimentServiceUnarchiveExperimentDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type ExperimentServiceUnarchiveExperimentDefault struct {
	_statusCode int

	Payload *experiment_model.GooglerpcStatus
}

// IsSuccess returns true when this experiment service unarchive experiment default response has a 2xx status code
func (o *ExperimentServiceUnarchiveExperimentDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this experiment service unarchive experiment default response has a 3xx status code
func (o *ExperimentServiceUnarchiveExperimentDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this experiment service unarchive experiment default response has a 4xx status code
func (o *ExperimentServiceUnarchiveExperimentDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this experiment service unarchive experiment default response has a 5xx status code
func (o *ExperimentServiceUnarchiveExperimentDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this experiment service unarchive experiment default response a status code equal to that given
func (o *ExperimentServiceUnarchiveExperimentDefault) IsCode(code int) bool {
	return o._statusCode == code
}

// Code gets the status code for the experiment service unarchive experiment default response
func (o *ExperimentServiceUnarchiveExperimentDefault) Code() int {
	return o._statusCode
}

func (o *ExperimentServiceUnarchiveExperimentDefault) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /apis/v2beta1/experiments/{experiment_id}:unarchive][%d] ExperimentService_UnarchiveExperiment default %s", o._statusCode, payload)
}

func (o *ExperimentServiceUnarchiveExperimentDefault) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[POST /apis/v2beta1/experiments/{experiment_id}:unarchive][%d] ExperimentService_UnarchiveExperiment default %s", o._statusCode, payload)
}

func (o *ExperimentServiceUnarchiveExperimentDefault) GetPayload() *experiment_model.GooglerpcStatus {
	return o.Payload
}

func (o *ExperimentServiceUnarchiveExperimentDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.GooglerpcStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
