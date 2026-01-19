// Package workflows contains commands for the n8n-cli workflows.
package workflows

import (
	"fmt"

	rootcmd "github.com/edenreich/n8n-cli/cmd"
	"github.com/edenreich/n8n-cli/n8n"
)

func resolveWorkflowIDByName(client n8n.ClientInterface, name string) (string, error) {
	workflowList, err := client.GetWorkflows()
	if err != nil {
		return "", fmt.Errorf("error fetching workflows: %w", err)
	}

	if workflowList == nil || workflowList.Data == nil || len(*workflowList.Data) == 0 {
		return "", fmt.Errorf("no workflows found in n8n instance")
	}

	return rootcmd.FindWorkflow(name, *workflowList.Data)
}
