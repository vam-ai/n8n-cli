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
package workflows

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	rootcmd "github.com/edenreich/n8n-cli/cmd"
	"github.com/edenreich/n8n-cli/n8n"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Output format constants
const (
	formatTable = "table"
	formatJSON  = "json"
	formatYAML  = "yaml"
)

var (
	// outputFormat defines the output format flag for the list command
	outputFormat string
	// sortOrder defines the sort order for last updated
	sortOrder string
)

// listCmd represents the list command
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List JSON workflows in n8n instance",
	Long:  `List command fetches and lists JSON workflows from a specified n8n instance.`,
	Args:  cobra.ExactArgs(0),
	RunE:  listWorkflows,
}

func init() {
	ListCmd.Flags().StringVarP(&outputFormat, "output", "o", formatTable, "Output format: table, json, or yaml")
	ListCmd.Flags().StringVar(&sortOrder, "order", "asc", "Sort order for lastUpdated: asc or desc")
	rootcmd.GetWorkflowsCmd().AddCommand(ListCmd)
}

// printWorkflowTable prints the workflows in a table format
func printWorkflowTable(cmd *cobra.Command, workflows []n8n.Workflow) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	_, err := fmt.Fprintln(w, "ID\tNAME\tACTIVE\tLAST_UPDATED")
	if err != nil {
		cmd.Println("Error printing workflow table:", err)
		return
	}
	for _, workflow := range workflows {
		var id, active, lastUpdated string
		if workflow.Id != nil {
			id = *workflow.Id
		} else {
			id = "N/A"
		}

		if workflow.Active != nil && *workflow.Active {
			active = "Yes"
		} else {
			active = "No"
		}

		if workflow.UpdatedAt != nil {
			lastUpdated = workflow.UpdatedAt.Format(time.RFC3339)
		} else {
			lastUpdated = "N/A"
		}

		_, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, workflow.Name, active, lastUpdated)
		if err != nil {
			cmd.Println("Error printing workflow table:", err)
			return
		}
	}
	err = w.Flush()
	if err != nil {
		cmd.Println("Error flushing workflow table:", err)
		return
	}
}

// printWorkflowJSON prints the workflows in JSON format
func printWorkflowJSON(cmd *cobra.Command, workflows []n8n.Workflow) error {
	jsonData, err := json.MarshalIndent(workflows, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling workflows to JSON: %w", err)
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(jsonData))
	return err
}

// printWorkflowYAML prints the workflows in YAML format
func printWorkflowYAML(cmd *cobra.Command, workflows []n8n.Workflow) error {
	yamlData, err := yaml.Marshal(workflows)
	if err != nil {
		return fmt.Errorf("error marshaling workflows to YAML: %w", err)
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(yamlData))
	return err
}

// listWorkflows fetches and lists workflows from the n8n instance
func listWorkflows(cmd *cobra.Command, args []string) error {
	apiKey := viper.Get("api_key").(string)
	instanceURL := viper.Get("instance_url").(string)

	client := n8n.NewClient(instanceURL, apiKey)

	workflowList, err := client.GetWorkflows()
	if err != nil {
		return err
	}

	if workflowList == nil || workflowList.Data == nil || len(*workflowList.Data) == 0 {
		cmd.Println("No workflows found")
		return nil
	}

	workflows := *workflowList.Data
	order := strings.ToLower(sortOrder)
	if order == "" {
		order = "asc"
	}
	if order != "asc" && order != "desc" {
		return fmt.Errorf("unsupported sort order: %s. Supported orders: asc, desc", sortOrder)
	}

	sort.Slice(workflows, func(i, j int) bool {
		left := workflows[i].UpdatedAt
		right := workflows[j].UpdatedAt
		if left == nil && right == nil {
			return workflows[i].Name < workflows[j].Name
		}
		if left == nil {
			return order == "asc"
		}
		if right == nil {
			return order != "asc"
		}
		if left.Equal(*right) {
			return workflows[i].Name < workflows[j].Name
		}
		if order == "asc" {
			return left.Before(*right)
		}
		return left.After(*right)
	})

	format := strings.ToLower(outputFormat)

	switch format {
	case formatJSON:
		return printWorkflowJSON(cmd, workflows)
	case formatYAML:
		return printWorkflowYAML(cmd, workflows)
	case formatTable:
		printWorkflowTable(cmd, workflows)
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s. Supported formats: table, json, yaml", outputFormat)
	}
}
