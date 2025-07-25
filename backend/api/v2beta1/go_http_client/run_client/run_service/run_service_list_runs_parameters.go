// Code generated by go-swagger; DO NOT EDIT.

package run_service

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
	"github.com/go-openapi/swag"
)

// NewRunServiceListRunsParams creates a new RunServiceListRunsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewRunServiceListRunsParams() *RunServiceListRunsParams {
	return &RunServiceListRunsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewRunServiceListRunsParamsWithTimeout creates a new RunServiceListRunsParams object
// with the ability to set a timeout on a request.
func NewRunServiceListRunsParamsWithTimeout(timeout time.Duration) *RunServiceListRunsParams {
	return &RunServiceListRunsParams{
		timeout: timeout,
	}
}

// NewRunServiceListRunsParamsWithContext creates a new RunServiceListRunsParams object
// with the ability to set a context for a request.
func NewRunServiceListRunsParamsWithContext(ctx context.Context) *RunServiceListRunsParams {
	return &RunServiceListRunsParams{
		Context: ctx,
	}
}

// NewRunServiceListRunsParamsWithHTTPClient creates a new RunServiceListRunsParams object
// with the ability to set a custom HTTPClient for a request.
func NewRunServiceListRunsParamsWithHTTPClient(client *http.Client) *RunServiceListRunsParams {
	return &RunServiceListRunsParams{
		HTTPClient: client,
	}
}

/*
RunServiceListRunsParams contains all the parameters to send to the API endpoint

	for the run service list runs operation.

	Typically these are written to a http.Request.
*/
type RunServiceListRunsParams struct {

	/* ExperimentID.

	   The ID of the parent experiment. If empty, response includes runs across all experiments.
	*/
	ExperimentID *string

	/* Filter.

	     A url-encoded, JSON-serialized Filter protocol buffer (see
	[filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)).
	*/
	Filter *string

	/* Namespace.

	   Optional input field. Filters based on the namespace.
	*/
	Namespace *string

	/* PageSize.

	     The number of runs to be listed per page. If there are more runs than this
	number, the response message will contain a nextPageToken field you can use
	to fetch the next page.

	     Format: int32
	*/
	PageSize *int32

	/* PageToken.

	     A page token to request the next page of results. The token is acquired
	from the nextPageToken field of the response from the previous
	ListRuns call or can be omitted when fetching the first page.
	*/
	PageToken *string

	/* SortBy.

	     Can be format of "field_name", "field_name asc" or "field_name desc"
	(Example, "name asc" or "id desc"). Ascending by default.
	*/
	SortBy *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the run service list runs params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RunServiceListRunsParams) WithDefaults() *RunServiceListRunsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the run service list runs params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RunServiceListRunsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the run service list runs params
func (o *RunServiceListRunsParams) WithTimeout(timeout time.Duration) *RunServiceListRunsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the run service list runs params
func (o *RunServiceListRunsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the run service list runs params
func (o *RunServiceListRunsParams) WithContext(ctx context.Context) *RunServiceListRunsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the run service list runs params
func (o *RunServiceListRunsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the run service list runs params
func (o *RunServiceListRunsParams) WithHTTPClient(client *http.Client) *RunServiceListRunsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the run service list runs params
func (o *RunServiceListRunsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithExperimentID adds the experimentID to the run service list runs params
func (o *RunServiceListRunsParams) WithExperimentID(experimentID *string) *RunServiceListRunsParams {
	o.SetExperimentID(experimentID)
	return o
}

// SetExperimentID adds the experimentId to the run service list runs params
func (o *RunServiceListRunsParams) SetExperimentID(experimentID *string) {
	o.ExperimentID = experimentID
}

// WithFilter adds the filter to the run service list runs params
func (o *RunServiceListRunsParams) WithFilter(filter *string) *RunServiceListRunsParams {
	o.SetFilter(filter)
	return o
}

// SetFilter adds the filter to the run service list runs params
func (o *RunServiceListRunsParams) SetFilter(filter *string) {
	o.Filter = filter
}

// WithNamespace adds the namespace to the run service list runs params
func (o *RunServiceListRunsParams) WithNamespace(namespace *string) *RunServiceListRunsParams {
	o.SetNamespace(namespace)
	return o
}

// SetNamespace adds the namespace to the run service list runs params
func (o *RunServiceListRunsParams) SetNamespace(namespace *string) {
	o.Namespace = namespace
}

// WithPageSize adds the pageSize to the run service list runs params
func (o *RunServiceListRunsParams) WithPageSize(pageSize *int32) *RunServiceListRunsParams {
	o.SetPageSize(pageSize)
	return o
}

// SetPageSize adds the pageSize to the run service list runs params
func (o *RunServiceListRunsParams) SetPageSize(pageSize *int32) {
	o.PageSize = pageSize
}

// WithPageToken adds the pageToken to the run service list runs params
func (o *RunServiceListRunsParams) WithPageToken(pageToken *string) *RunServiceListRunsParams {
	o.SetPageToken(pageToken)
	return o
}

// SetPageToken adds the pageToken to the run service list runs params
func (o *RunServiceListRunsParams) SetPageToken(pageToken *string) {
	o.PageToken = pageToken
}

// WithSortBy adds the sortBy to the run service list runs params
func (o *RunServiceListRunsParams) WithSortBy(sortBy *string) *RunServiceListRunsParams {
	o.SetSortBy(sortBy)
	return o
}

// SetSortBy adds the sortBy to the run service list runs params
func (o *RunServiceListRunsParams) SetSortBy(sortBy *string) {
	o.SortBy = sortBy
}

// WriteToRequest writes these params to a swagger request
func (o *RunServiceListRunsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.ExperimentID != nil {

		// query param experiment_id
		var qrExperimentID string

		if o.ExperimentID != nil {
			qrExperimentID = *o.ExperimentID
		}
		qExperimentID := qrExperimentID
		if qExperimentID != "" {

			if err := r.SetQueryParam("experiment_id", qExperimentID); err != nil {
				return err
			}
		}
	}

	if o.Filter != nil {

		// query param filter
		var qrFilter string

		if o.Filter != nil {
			qrFilter = *o.Filter
		}
		qFilter := qrFilter
		if qFilter != "" {

			if err := r.SetQueryParam("filter", qFilter); err != nil {
				return err
			}
		}
	}

	if o.Namespace != nil {

		// query param namespace
		var qrNamespace string

		if o.Namespace != nil {
			qrNamespace = *o.Namespace
		}
		qNamespace := qrNamespace
		if qNamespace != "" {

			if err := r.SetQueryParam("namespace", qNamespace); err != nil {
				return err
			}
		}
	}

	if o.PageSize != nil {

		// query param page_size
		var qrPageSize int32

		if o.PageSize != nil {
			qrPageSize = *o.PageSize
		}
		qPageSize := swag.FormatInt32(qrPageSize)
		if qPageSize != "" {

			if err := r.SetQueryParam("page_size", qPageSize); err != nil {
				return err
			}
		}
	}

	if o.PageToken != nil {

		// query param page_token
		var qrPageToken string

		if o.PageToken != nil {
			qrPageToken = *o.PageToken
		}
		qPageToken := qrPageToken
		if qPageToken != "" {

			if err := r.SetQueryParam("page_token", qPageToken); err != nil {
				return err
			}
		}
	}

	if o.SortBy != nil {

		// query param sort_by
		var qrSortBy string

		if o.SortBy != nil {
			qrSortBy = *o.SortBy
		}
		qSortBy := qrSortBy
		if qSortBy != "" {

			if err := r.SetQueryParam("sort_by", qSortBy); err != nil {
				return err
			}
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
