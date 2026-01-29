// Package unit contains unit tests for the n8n-cli
package unit

import (
	"bytes"
	"errors"
	"testing"

	"github.com/edenreich/n8n-cli/cmd/workflows"
	"github.com/edenreich/n8n-cli/n8n/clientfakes"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestDeleteCommandExecute(t *testing.T) {
	testCases := []struct {
		name           string
		workflowID     string
		mockReturnErr  error
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "Successfully deletes workflow",
			workflowID:     "123",
			mockReturnErr:  nil,
			expectedOutput: "Workflow with ID 123 has been deleted successfully\n",
			expectError:    false,
		},
		{
			name:           "Returns error when API call fails",
			workflowID:     "789",
			mockReturnErr:  errors.New("API error"),
			expectedOutput: "Error deleting workflow: API error\n",
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := &clientfakes.FakeClientInterface{}

			fakeClient.DeleteWorkflowReturns(tc.mockReturnErr)

			outBuf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)

			cmd := &cobra.Command{}
			cmd.SetOut(outBuf)
			cmd.SetErr(errBuf)

			origRunE := workflows.DeleteCmd.RunE
			defer func() {
				workflows.DeleteCmd.RunE = origRunE
			}()

			testRunE := func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return errors.New("this command requires a workflow ID")
				}

				workflowID := args[0]
				err := fakeClient.DeleteWorkflow(workflowID)

				if err != nil {
					cmd.PrintErrf("Error deleting workflow: %v\n", err)
					return err
				}

				cmd.Printf("Workflow with ID %s has been deleted successfully\n", workflowID)

				return nil
			}

			workflows.DeleteCmd.RunE = testRunE

			err := workflows.DeleteCmd.RunE(cmd, []string{tc.workflowID})

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			output := outBuf.String() + errBuf.String()
			assert.Equal(t, tc.expectedOutput, output)

			assert.Equal(t, 1, fakeClient.DeleteWorkflowCallCount())
			passedWorkflowID := fakeClient.DeleteWorkflowArgsForCall(0)
			assert.Equal(t, tc.workflowID, passedWorkflowID)
		})
	}
}

func TestDeleteCmd_NoArgs(t *testing.T) {
	cmd := &cobra.Command{}

	runE := workflows.DeleteCmd.RunE

	err := runE(cmd, []string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "this command requires a workflow ID")
}
