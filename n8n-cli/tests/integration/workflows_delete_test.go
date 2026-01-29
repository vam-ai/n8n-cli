// Package integration contains integration tests for the n8n-cli
package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rootcmd "github.com/edenreich/n8n-cli/cmd"
	"github.com/edenreich/n8n-cli/config"
	"github.com/edenreich/n8n-cli/n8n"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// setupDeleteWorkflowTest creates a test server and configures the test environment
func setupDeleteWorkflowTest(t *testing.T) (*httptest.Server, map[string]*n8n.Workflow, func()) {
	mockWorkflows := map[string]*n8n.Workflow{
		"123": {
			Id:   stringPtr("123"),
			Name: "Test Workflow 1",
		},
		"456": {
			Id:   stringPtr("456"),
			Name: "Test Workflow 2",
		},
		"nonexistent": nil,
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-N8N-API-KEY")
		if apiKey != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprintln(w, `{"error": "Unauthorized"}`)
			return
		}

		if r.Method == http.MethodDelete {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) == 5 && parts[3] == "workflows" {
				workflowID := parts[4]

				workflow, exists := mockWorkflows[workflowID]
				if !exists || workflow == nil {
					w.WriteHeader(http.StatusNotFound)
					_, _ = fmt.Fprintln(w, `{"error": "Workflow not found"}`)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(workflow)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = fmt.Fprintln(w, `{"error": "Failed to encode response"}`)
					return
				}
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error": "Not found"}`)
	}))

	viper.Reset()
	viper.Set("api_key", "test-api-key")
	viper.Set("instance_url", mockServer.URL)
	config.Initialize()

	cleanup := func() {
		mockServer.Close()
		viper.Reset()
	}

	return mockServer, mockWorkflows, cleanup
}

// TestDeleteWorkflow tests the delete workflow command
func TestDeleteWorkflow(t *testing.T) {
	_, _, cleanup := setupDeleteWorkflowTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "Successfully delete existing workflow",
			args:           []string{"delete", "123"},
			expectedOutput: "Workflow with ID 123 has been deleted successfully",
			expectedError:  false,
		},
		{
			name:           "Fail with non-existent workflow",
			args:           []string{"delete", "nonexistent"},
			expectedOutput: "",
			expectedError:  true,
			errorContains:  "Workflow not found",
		},
		{
			name:           "Missing workflow ID",
			args:           []string{"delete"},
			expectedOutput: "",
			expectedError:  true,
			errorContains:  "accepts 1 arg(s), received 0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := executeCommand(t, rootcmd.GetWorkflowsCmd(), tc.args...)

			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.True(t, strings.Contains(stderr, tc.errorContains) || (err != nil && strings.Contains(err.Error(), tc.errorContains)),
						"Expected error containing '%s', got '%s' (error: '%v')", tc.errorContains, stderr, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, stdout, tc.expectedOutput)
			}
		})
	}
}
