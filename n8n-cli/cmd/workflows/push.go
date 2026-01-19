/*
Copyright Ac 2025 Eden Reich

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package workflows

import (
	"fmt"
	"path/filepath"

	rootcmd "github.com/edenreich/n8n-cli/cmd"
	"github.com/edenreich/n8n-cli/n8n"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push a local workflow file to n8n",
	Long: `Push command uploads a single workflow file to an n8n instance.

Provide a workflow name or file path. If the name exists on the server, the
workflow will be updated. If it does not exist, the local file ID (if any) is used.`,
	Args: cobra.ExactArgs(0),
	RunE: PushWorkflow,
}

func init() {
	pushCmd.Flags().StringP("directory", "d", "", "Directory containing workflow files")
	pushCmd.Flags().StringP("file", "f", "", "Workflow name or file path")
	pushCmd.Flags().String("id", "", "Workflow ID to push")
	pushCmd.Flags().StringP("name", "n", "", "Workflow name to push")
	pushCmd.Flags().Bool("dry-run", false, "Show what would be uploaded without making changes")
	rootcmd.GetWorkflowsCmd().AddCommand(pushCmd)

	// nolint:errcheck
	pushCmd.MarkFlagFilename("file", "json", "yaml", "yml")
}

// PushWorkflow pushes a single workflow file by ID or name.
func PushWorkflow(cmd *cobra.Command, args []string) error {
	cmd.Println("Pushing workflow...")
	directory, _ := cmd.Flags().GetString("directory")
	fileFlag, _ := cmd.Flags().GetString("file")
	workflowID, _ := cmd.Flags().GetString("id")
	workflowName, _ := cmd.Flags().GetString("name")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if workflowID != "" && workflowName != "" {
		return fmt.Errorf("use either --id or --name, not both")
	}

	if fileFlag != "" && workflowName == "" && workflowID == "" && !looksLikeFilePath(fileFlag) {
		workflowName = fileFlag
		fileFlag = ""
	}

	filePath := fileFlag
	if filePath != "" && !looksLikeFilePath(filePath) {
		workflowName = filePath
		filePath = ""
	}

	if filePath == "" && workflowName == "" {
		return fmt.Errorf("workflow name or file is required")
	}

	if filePath == "" {
		localFilePath, localWorkflow, localFound, err := findLocalWorkflowByName(directory, workflowName)
		if err != nil {
			return err
		}
		if !localFound || localFilePath == "" {
			return fmt.Errorf("workflow '%s' not found in %s", workflowName, directory)
		}
		filePath = localFilePath
		if workflowName == "" {
			workflowName = localWorkflow.Name
		}
	}

	if err := validateWorkflowFileExtension(filePath); err != nil {
		return err
	}

	apiKey := viper.Get("api_key").(string)
	instanceURL := viper.Get("instance_url").(string)
	client := n8n.NewClient(instanceURL, apiKey)

	workflow, err := readWorkflowFromFile(filePath)
	if err != nil {
		return err
	}

	if workflowID != "" {
		workflow.Id = &workflowID
	} else if workflowName != "" {
		resolvedID, resolveErr := resolveWorkflowIDByName(client, workflowName)
		if resolveErr == nil {
			workflow.Id = &resolvedID
		} else if !isWorkflowNameNotFound(resolveErr) {
			return resolveErr
		}
	}

	filename := filepath.Base(filePath)
	result, err := processWorkflowPayload(client, cmd, &workflow, filename, filePath, dryRun)
	if err != nil {
		return err
	}

	if result.WorkflowID != "" {
		cmd.Printf("Workflow '%s' synced (ID: %s) from %s\n", workflow.Name, result.WorkflowID, filename)
	}

	return nil
}
