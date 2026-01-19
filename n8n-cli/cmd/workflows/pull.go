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
	"strings"

	rootcmd "github.com/edenreich/n8n-cli/cmd"
	"github.com/edenreich/n8n-cli/n8n"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull a workflow from n8n into a local file",
	Long: `Pull command fetches a single workflow from an n8n instance and writes it to a local file.

Provide a workflow name or ID. If the name is not found on the server, the CLI
will check the local directory for a matching workflow file and use its ID.`,
	Args: cobra.ExactArgs(0),
	RunE: PullWorkflow,
}

func init() {
	pullCmd.Flags().StringP("directory", "d", "", "Directory containing workflow files")
	pullCmd.Flags().StringP("file", "f", "", "Workflow name or file path")
	pullCmd.Flags().String("id", "", "Workflow ID to pull")
	pullCmd.Flags().StringP("name", "n", "", "Workflow name to pull")
	pullCmd.Flags().StringP("output", "o", "json", "Output format for workflow file (json or yaml)")
	pullCmd.Flags().Bool("no-truncate", false, "Include all fields in output files, including null and optional fields")
	pullCmd.Flags().Bool("dry-run", false, "Show what would be updated without making changes")
	rootcmd.GetWorkflowsCmd().AddCommand(pullCmd)

	// nolint:errcheck
	pullCmd.MarkFlagFilename("file", "json", "yaml", "yml")
}

// PullWorkflow pulls a single workflow by ID or name and writes it to a file.
func PullWorkflow(cmd *cobra.Command, args []string) error {
	cmd.Println("Pulling workflow...")
	directory, _ := cmd.Flags().GetString("directory")
	fileFlag, _ := cmd.Flags().GetString("file")
	workflowID, _ := cmd.Flags().GetString("id")
	workflowName, _ := cmd.Flags().GetString("name")
	output, _ := cmd.Flags().GetString("output")
	noTruncate, _ := cmd.Flags().GetBool("no-truncate")
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

	if workflowID == "" && workflowName == "" && filePath == "" {
		return fmt.Errorf("workflow id, name, or file is required")
	}

	apiKey := viper.Get("api_key").(string)
	instanceURL := viper.Get("instance_url").(string)
	client := n8n.NewClient(instanceURL, apiKey)

	localFilePath, localWorkflow, localFound, err := findLocalWorkflowByName(directory, workflowName)
	if err != nil {
		return err
	}

	if workflowID == "" && workflowName != "" {
		resolvedID, resolveErr := resolveWorkflowIDByName(client, workflowName)
		if resolveErr != nil {
			if !isWorkflowNameNotFound(resolveErr) {
				return resolveErr
			}
			if localFound && localWorkflow.Id != nil && *localWorkflow.Id != "" {
				workflowID = *localWorkflow.Id
			}
		} else {
			workflowID = resolvedID
		}
	}

	if workflowID == "" && filePath != "" {
		if id, err := ExtractWorkflowIDFromFile(filePath); err == nil && id != "" {
			workflowID = id
		} else if workflowName == "" {
			if workflow, err := readWorkflowFromFile(filePath); err == nil && workflow.Name != "" {
				workflowName = workflow.Name
			}
		}
	}

	if workflowID == "" && workflowName == "" {
		return fmt.Errorf("workflow id or name is required")
	}

	if workflowID == "" {
		return fmt.Errorf("workflow '%s' not found on server and no local ID in %s", workflowName, directory)
	}

	workflow, err := client.GetWorkflow(workflowID)
	if err != nil {
		return fmt.Errorf("error fetching workflow: %w", err)
	}

	if workflowName == "" && workflow != nil && workflow.Name != "" {
		workflowName = workflow.Name
	}

	if filePath == "" {
		if directory != "" {
			if localFilesByID, err := extractLocalWorkflows(directory); err != nil {
				return err
			} else if existingPath, ok := localFilesByID[workflowID]; ok {
				filePath = existingPath
			}
		}

		if filePath == "" && localFound && localFilePath != "" {
			filePath = localFilePath
		} else if filePath == "" {
			if directory == "" {
				return fmt.Errorf("directory is required when no file path is provided")
			}
			extension := ".json"
			if strings.EqualFold(output, "yaml") || strings.EqualFold(output, "yml") {
				extension = ".yaml"
			}
			filePath = filepath.Join(directory, rootcmd.SanitizeFilename(workflowName)+extension)
		}
	}

	if output != "" {
		extension := ".json"
		if strings.EqualFold(output, "yaml") || strings.EqualFold(output, "yml") {
			extension = ".yaml"
		}
		filePath = strings.TrimSuffix(filePath, filepath.Ext(filePath)) + extension
	}

	if err := validateWorkflowFileExtension(filePath); err != nil {
		return err
	}

	parentDir := filepath.Dir(filePath)
	if parentDir != "." {
		if err := ensureDirectoryExists(cmd, parentDir, dryRun); err != nil {
			return err
		}
	}

	minimal := !noTruncate
	return refreshWorkflowToFile(cmd, *workflow, filePath, dryRun, minimal)
}

func isWorkflowNameNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "workflow with name") && strings.Contains(msg, "not found")
}
