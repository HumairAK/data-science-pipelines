// Code generated by go-swagger; DO NOT EDIT.

package pipeline_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewPipelineServiceGetPipelineVersionParams creates a new PipelineServiceGetPipelineVersionParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewPipelineServiceGetPipelineVersionParams() *PipelineServiceGetPipelineVersionParams {
	return &PipelineServiceGetPipelineVersionParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewPipelineServiceGetPipelineVersionParamsWithTimeout creates a new PipelineServiceGetPipelineVersionParams object
// with the ability to set a timeout on a request.
func NewPipelineServiceGetPipelineVersionParamsWithTimeout(timeout time.Duration) *PipelineServiceGetPipelineVersionParams {
	return &PipelineServiceGetPipelineVersionParams{
		timeout: timeout,
	}
}

// NewPipelineServiceGetPipelineVersionParamsWithContext creates a new PipelineServiceGetPipelineVersionParams object
// with the ability to set a context for a request.
func NewPipelineServiceGetPipelineVersionParamsWithContext(ctx context.Context) *PipelineServiceGetPipelineVersionParams {
	return &PipelineServiceGetPipelineVersionParams{
		Context: ctx,
	}
}

// NewPipelineServiceGetPipelineVersionParamsWithHTTPClient creates a new PipelineServiceGetPipelineVersionParams object
// with the ability to set a custom HTTPClient for a request.
func NewPipelineServiceGetPipelineVersionParamsWithHTTPClient(client *http.Client) *PipelineServiceGetPipelineVersionParams {
	return &PipelineServiceGetPipelineVersionParams{
		HTTPClient: client,
	}
}

/*
PipelineServiceGetPipelineVersionParams contains all the parameters to send to the API endpoint

	for the pipeline service get pipeline version operation.

	Typically these are written to a http.Request.
*/
type PipelineServiceGetPipelineVersionParams struct {

	/* PipelineID.

	   Required input. ID of the parent pipeline.
	*/
	PipelineID string

	/* PipelineVersionID.

	   Required input. ID of the pipeline version to be retrieved.
	*/
	PipelineVersionID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the pipeline service get pipeline version params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *PipelineServiceGetPipelineVersionParams) WithDefaults() *PipelineServiceGetPipelineVersionParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the pipeline service get pipeline version params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *PipelineServiceGetPipelineVersionParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) WithTimeout(timeout time.Duration) *PipelineServiceGetPipelineVersionParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) WithContext(ctx context.Context) *PipelineServiceGetPipelineVersionParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) WithHTTPClient(client *http.Client) *PipelineServiceGetPipelineVersionParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithPipelineID adds the pipelineID to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) WithPipelineID(pipelineID string) *PipelineServiceGetPipelineVersionParams {
	o.SetPipelineID(pipelineID)
	return o
}

// SetPipelineID adds the pipelineId to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) SetPipelineID(pipelineID string) {
	o.PipelineID = pipelineID
}

// WithPipelineVersionID adds the pipelineVersionID to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) WithPipelineVersionID(pipelineVersionID string) *PipelineServiceGetPipelineVersionParams {
	o.SetPipelineVersionID(pipelineVersionID)
	return o
}

// SetPipelineVersionID adds the pipelineVersionId to the pipeline service get pipeline version params
func (o *PipelineServiceGetPipelineVersionParams) SetPipelineVersionID(pipelineVersionID string) {
	o.PipelineVersionID = pipelineVersionID
}

// WriteToRequest writes these params to a swagger request
func (o *PipelineServiceGetPipelineVersionParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param pipeline_id
	if err := r.SetPathParam("pipeline_id", o.PipelineID); err != nil {
		return err
	}

	// path param pipeline_version_id
	if err := r.SetPathParam("pipeline_version_id", o.PipelineVersionID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
