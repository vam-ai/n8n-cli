/*
Copyright © 2025 Eden Reich

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
package cmd

import (
	"github.com/spf13/cobra"
)

// workflowsCmd represents the workflows command
var workflowsCmd = &cobra.Command{
	Use:   "workflows",
	Short: "Manage n8n workflows",
	Long: `The workflows command provides utilities to import, export, list, and 
synchronize n8n workflows between your local filesystem and n8n instances.

Usage:
  n8n workflows [flags]
  n8n workflows [command]

Available Commands:
  activate    Activate a workflow by ID
  deactivate  Deactivate a workflow by ID
  delete      Delete a workflow by ID
  executions  Get execution history for workflows
  list        List JSON workflows in n8n instance
  pull        Pull a workflow from n8n into a local file
  push        Push a local workflow file to n8n
  refresh     Refresh the state of workflows in the directory from n8n instance
  sync        Synchronize workflows between local files and n8n instance

Flags:
  -h, --help   help for workflows

Global Flags:
  -k, --api-key string   n8n API Key (env: N8N_API_KEY)
      --debug            Enable debug logging (env: DEBUG)
  -u, --url string       n8n instance URL (env: N8N_INSTANCE_URL) (default "http://localhost:5678")

Use "n8n workflows [command] --help" for more information about a command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(workflowsCmd)
}

// GetWorkflowsCmd returns the workflows command for other packages
func GetWorkflowsCmd() *cobra.Command {
	return workflowsCmd
}
