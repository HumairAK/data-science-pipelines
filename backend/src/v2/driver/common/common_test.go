package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isInputParameterChannel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{
			name:    "wellformed pipeline channel should produce no errors",
			input:   "{{$.inputs.parameters['pipelinechannel--someParameterName']}}",
			isValid: true,
		},
		{
			name:    "pipeline channel index should have quotes",
			input:   "{{$.inputs.parameters[pipelinechannel--someParameterName]}}",
			isValid: false,
		},
		{
			name:    "plain text as pipelinechannel of parameter type is invalid",
			input:   "randomtext",
			isValid: false,
		},
		{
			name:    "inputs should be prefixed with $.",
			input:   "{{inputs.parameters['pipelinechannel--someParameterName']}}",
			isValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, isInputParameterChannel(test.input), test.isValid)
		})
	}
}

func Test_extractInputParameterFromChannel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "standard parameter pipeline channel input",
			input:    "{{$.inputs.parameters['pipelinechannel--someParameterName']}}",
			expected: "pipelinechannel--someParameterName",
			wantErr:  false,
		},
		{
			name:     "a more complex parameter pipeline channel input",
			input:    "{{$.inputs.parameters['pipelinechannel--somePara-me_terName']}}",
			expected: "pipelinechannel--somePara-me_terName",
			wantErr:  false,
		},
		{
			name:    "invalid input should return err",
			input:   "invalidvalue",
			wantErr: true,
		},
		{
			name:    "invalid input should return err 2",
			input:   "pipelinechannel--somePara-me_terName",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := extractInputParameterFromChannel(test.input)
			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, actual, test.expected)
			}
		})
	}
}
