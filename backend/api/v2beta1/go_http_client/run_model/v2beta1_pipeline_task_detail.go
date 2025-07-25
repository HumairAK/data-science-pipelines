// Code generated by go-swagger; DO NOT EDIT.

package run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// V2beta1PipelineTaskDetail Runtime information of a task execution.
//
// swagger:model v2beta1PipelineTaskDetail
type V2beta1PipelineTaskDetail struct {

	// Sequence of dependen tasks.
	ChildTasks []*PipelineTaskDetailChildTask `json:"child_tasks"`

	// Creation time of a task.
	// Format: date-time
	CreateTime strfmt.DateTime `json:"create_time,omitempty"`

	// User specified name of a task that is defined in
	// [Pipeline.spec][].
	DisplayName string `json:"display_name,omitempty"`

	// Completion time of a task.
	// Format: date-time
	EndTime strfmt.DateTime `json:"end_time,omitempty"`

	// The error that occurred during task execution.
	// Only populated when the task is in FAILED or CANCELED state.
	Error *GooglerpcStatus `json:"error,omitempty"`

	// Execution id of the corresponding entry in ML metadata store.
	ExecutionID string `json:"execution_id,omitempty"`

	// Execution information of a task.
	ExecutorDetail *V2beta1PipelineTaskExecutorDetail `json:"executor_detail,omitempty"`

	// Input artifacts of the task.
	Inputs map[string]V2beta1ArtifactList `json:"inputs,omitempty"`

	// Output artifacts of the task.
	Outputs map[string]V2beta1ArtifactList `json:"outputs,omitempty"`

	// ID of the parent task if the task is within a component scope.
	// Empty if the task is at the root level.
	ParentTaskID string `json:"parent_task_id,omitempty"`

	// Name of the corresponding pod assigned by the orchestration engine.
	// Also known as node_id.
	PodName string `json:"pod_name,omitempty"`

	// ID of the parent run.
	RunID string `json:"run_id,omitempty"`

	// Starting time of a task.
	// Format: date-time
	StartTime strfmt.DateTime `json:"start_time,omitempty"`

	// Runtime state of a task.
	State *V2beta1RuntimeState `json:"state,omitempty"`

	// A sequence of task statuses. This field keeps a record
	// of state transitions.
	StateHistory []*V2beta1RuntimeStatus `json:"state_history"`

	// System-generated ID of a task.
	TaskID string `json:"task_id,omitempty"`
}

// Validate validates this v2beta1 pipeline task detail
func (m *V2beta1PipelineTaskDetail) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateChildTasks(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCreateTime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateEndTime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateError(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateExecutorDetail(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateInputs(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOutputs(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStartTime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateState(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStateHistory(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V2beta1PipelineTaskDetail) validateChildTasks(formats strfmt.Registry) error {
	if swag.IsZero(m.ChildTasks) { // not required
		return nil
	}

	for i := 0; i < len(m.ChildTasks); i++ {
		if swag.IsZero(m.ChildTasks[i]) { // not required
			continue
		}

		if m.ChildTasks[i] != nil {
			if err := m.ChildTasks[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("child_tasks" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("child_tasks" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) validateCreateTime(formats strfmt.Registry) error {
	if swag.IsZero(m.CreateTime) { // not required
		return nil
	}

	if err := validate.FormatOf("create_time", "body", "date-time", m.CreateTime.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) validateEndTime(formats strfmt.Registry) error {
	if swag.IsZero(m.EndTime) { // not required
		return nil
	}

	if err := validate.FormatOf("end_time", "body", "date-time", m.EndTime.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) validateError(formats strfmt.Registry) error {
	if swag.IsZero(m.Error) { // not required
		return nil
	}

	if m.Error != nil {
		if err := m.Error.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("error")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("error")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) validateExecutorDetail(formats strfmt.Registry) error {
	if swag.IsZero(m.ExecutorDetail) { // not required
		return nil
	}

	if m.ExecutorDetail != nil {
		if err := m.ExecutorDetail.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("executor_detail")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("executor_detail")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) validateInputs(formats strfmt.Registry) error {
	if swag.IsZero(m.Inputs) { // not required
		return nil
	}

	for k := range m.Inputs {

		if err := validate.Required("inputs"+"."+k, "body", m.Inputs[k]); err != nil {
			return err
		}
		if val, ok := m.Inputs[k]; ok {
			if err := val.Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("inputs" + "." + k)
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("inputs" + "." + k)
				}
				return err
			}
		}

	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) validateOutputs(formats strfmt.Registry) error {
	if swag.IsZero(m.Outputs) { // not required
		return nil
	}

	for k := range m.Outputs {

		if err := validate.Required("outputs"+"."+k, "body", m.Outputs[k]); err != nil {
			return err
		}
		if val, ok := m.Outputs[k]; ok {
			if err := val.Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("outputs" + "." + k)
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("outputs" + "." + k)
				}
				return err
			}
		}

	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) validateStartTime(formats strfmt.Registry) error {
	if swag.IsZero(m.StartTime) { // not required
		return nil
	}

	if err := validate.FormatOf("start_time", "body", "date-time", m.StartTime.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) validateState(formats strfmt.Registry) error {
	if swag.IsZero(m.State) { // not required
		return nil
	}

	if m.State != nil {
		if err := m.State.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("state")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("state")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) validateStateHistory(formats strfmt.Registry) error {
	if swag.IsZero(m.StateHistory) { // not required
		return nil
	}

	for i := 0; i < len(m.StateHistory); i++ {
		if swag.IsZero(m.StateHistory[i]) { // not required
			continue
		}

		if m.StateHistory[i] != nil {
			if err := m.StateHistory[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("state_history" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("state_history" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this v2beta1 pipeline task detail based on the context it is used
func (m *V2beta1PipelineTaskDetail) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateChildTasks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateError(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateExecutorDetail(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateInputs(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOutputs(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateState(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateStateHistory(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V2beta1PipelineTaskDetail) contextValidateChildTasks(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.ChildTasks); i++ {

		if m.ChildTasks[i] != nil {

			if swag.IsZero(m.ChildTasks[i]) { // not required
				return nil
			}

			if err := m.ChildTasks[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("child_tasks" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("child_tasks" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) contextValidateError(ctx context.Context, formats strfmt.Registry) error {

	if m.Error != nil {

		if swag.IsZero(m.Error) { // not required
			return nil
		}

		if err := m.Error.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("error")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("error")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) contextValidateExecutorDetail(ctx context.Context, formats strfmt.Registry) error {

	if m.ExecutorDetail != nil {

		if swag.IsZero(m.ExecutorDetail) { // not required
			return nil
		}

		if err := m.ExecutorDetail.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("executor_detail")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("executor_detail")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) contextValidateInputs(ctx context.Context, formats strfmt.Registry) error {

	for k := range m.Inputs {

		if val, ok := m.Inputs[k]; ok {
			if err := val.ContextValidate(ctx, formats); err != nil {
				return err
			}
		}

	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) contextValidateOutputs(ctx context.Context, formats strfmt.Registry) error {

	for k := range m.Outputs {

		if val, ok := m.Outputs[k]; ok {
			if err := val.ContextValidate(ctx, formats); err != nil {
				return err
			}
		}

	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) contextValidateState(ctx context.Context, formats strfmt.Registry) error {

	if m.State != nil {

		if swag.IsZero(m.State) { // not required
			return nil
		}

		if err := m.State.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("state")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("state")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1PipelineTaskDetail) contextValidateStateHistory(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.StateHistory); i++ {

		if m.StateHistory[i] != nil {

			if swag.IsZero(m.StateHistory[i]) { // not required
				return nil
			}

			if err := m.StateHistory[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("state_history" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("state_history" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *V2beta1PipelineTaskDetail) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V2beta1PipelineTaskDetail) UnmarshalBinary(b []byte) error {
	var res V2beta1PipelineTaskDetail
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
