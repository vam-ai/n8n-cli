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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rootcmd "github.com/edenreich/n8n-cli/cmd"
	"github.com/edenreich/n8n-cli/n8n"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// refreshCmd represents the refresh command
var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the state of workflows in the directory from n8n instance",
	Long: `Refresh command fetches and updates the state of workflows in the directory from a specified n8n instance.
By default, only workflows that already exist in the directory will be refreshed. Use the --all flag to refresh all workflows.

For single files, use --file with --id or --name to refresh one workflow.`,
	Args: cobra.ExactArgs(0),
	RunE: RefreshWorkflows,
}

func init() {
	refreshCmd.Flags().StringP("directory", "d", "", "Directory containing workflow files (JSON/YAML)")
	refreshCmd.Flags().StringP("file", "f", "", "Single workflow file path to refresh (JSON/YAML)")
	refreshCmd.Flags().Bool("dry-run", false, "Show what would be updated without making changes")
	refreshCmd.Flags().Bool("overwrite", false, "Overwrite existing files even if they have a different name")
	refreshCmd.Flags().StringP("output", "o", "json", "Output format for new workflow files (json or yaml)")
	refreshCmd.Flags().Bool("no-truncate", false, "Include all fields in output files, including null and optional fields")
	refreshCmd.Flags().Bool("all", false, "Refresh all workflows from n8n instance, not just those in the directory")
	refreshCmd.Flags().String("id", "", "Workflow ID to refresh (used with --file)")
	refreshCmd.Flags().String("name", "", "Workflow name to refresh (used with --file)")
	rootcmd.GetWorkflowsCmd().AddCommand(refreshCmd)

	// nolint:errcheck
	refreshCmd.MarkFlagFilename("file", "json", "yaml", "yml")
}

// RefreshWorkflows refreshes workflow files from n8n instance
func RefreshWorkflows(cmd *cobra.Command, args []string) error {
	cmd.Println("Refreshing workflows...")
	directory, _ := cmd.Flags().GetString("directory")
	filePath, _ := cmd.Flags().GetString("file")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	output, _ := cmd.Flags().GetString("output")
	noTruncate, _ := cmd.Flags().GetBool("no-truncate")
	all, _ := cmd.Flags().GetBool("all")
	workflowID, _ := cmd.Flags().GetString("id")
	workflowName, _ := cmd.Flags().GetString("name")

	if filePath != "" && directory != "" {
		return fmt.Errorf("use either --file or --directory, not both")
	}

	if filePath == "" && directory == "" {
		return fmt.Errorf("directory or file is required")
	}

	if workflowID != "" && workflowName != "" {
		return fmt.Errorf("use either --id or --name, not both")
	}

	apiKey := viper.Get("api_key").(string)
	instanceURL := viper.Get("instance_url").(string)

	client := n8n.NewClient(instanceURL, apiKey)

	minimal := !noTruncate

	if filePath != "" {
		if err := validateWorkflowFileExtension(filePath); err != nil {
			return err
		}

		return RefreshSingleWorkflowWithClient(cmd, client, filePath, workflowID, workflowName, dryRun, minimal)
	}

	return RefreshWorkflowsWithClient(cmd, client, directory, dryRun, overwrite, output, minimal, all)
}

// RefreshWorkflowsWithClient is the testable version of RefreshWorkflows that accepts a client interface
func RefreshWorkflowsWithClient(cmd *cobra.Command, client n8n.ClientInterface, directory string, dryRun bool, overwrite bool, output string, minimal bool, all bool) error {
	if err := ensureDirectoryExists(cmd, directory, dryRun); err != nil {
		return err
	}

	localFiles, err := extractLocalWorkflows(directory)
	if err != nil {
		return err
	}

	if all || len(localFiles) == 0 {
		cmd.Println("Refreshing all workflows from n8n instance")

		workflowList, err := client.GetWorkflows()
		if err != nil {
			return fmt.Errorf("error fetching workflows: %w", err)
		}

		if workflowList == nil || workflowList.Data == nil || len(*workflowList.Data) == 0 {
			cmd.Println("No workflows found in n8n instance")
			return nil
		}

		for _, workflow := range *workflowList.Data {
			if err := processWorkflow(cmd, workflow, localFiles, directory, dryRun, overwrite, output, minimal); err != nil {
				return err
			}
		}
	} else {
		cmd.Println("Refreshing only workflows that exist in the directory")

		refreshed := 0
		for workflowID := range localFiles {
			workflow, err := client.GetWorkflow(workflowID)
			if err != nil {
				cmd.Printf("Warning: Could not fetch workflow with ID %s: %v\n", workflowID, err)
				continue
			}

			if err := processWorkflow(cmd, *workflow, localFiles, directory, dryRun, overwrite, output, minimal); err != nil {
				return err
			}
			refreshed++
		}

		if refreshed == 0 {
			cmd.Println("No workflows were refreshed. Either the local workflows don't exist in the n8n instance or there was an error fetching them. Try refresh --all and delete the local files you don't want to track.")
		}
	}

	cmd.Println("Workflow refresh completed successfully")
	return nil
}

// RefreshSingleWorkflowWithClient refreshes a single workflow file by ID or name.
func RefreshSingleWorkflowWithClient(cmd *cobra.Command, client n8n.ClientInterface, filePath string, workflowID string, workflowName string, dryRun bool, minimal bool) error {
	parentDir := filepath.Dir(filePath)
	if parentDir != "." {
		if err := ensureDirectoryExists(cmd, parentDir, dryRun); err != nil {
			return err
		}
	}

	if workflowID == "" && workflowName == "" {
		if _, err := os.Stat(filePath); err == nil {
			if id, err := ExtractWorkflowIDFromFile(filePath); err == nil && id != "" {
				workflowID = id
			} else if workflow, err := readWorkflowFromFile(filePath); err == nil {
				if workflow.Name != "" {
					workflowName = workflow.Name
				}
			}
		}
	}

	if workflowID == "" && workflowName == "" {
		return fmt.Errorf("workflow id or name is required when using --file")
	}

	var workflow *n8n.Workflow
	var err error

	if workflowID != "" {
		workflow, err = client.GetWorkflow(workflowID)
		if err != nil {
			return fmt.Errorf("error fetching workflow: %w", err)
		}
	} else {
		resolvedID, err := resolveWorkflowIDByName(client, workflowName)
		if err != nil {
			return err
		}

		workflow, err = client.GetWorkflow(resolvedID)
		if err != nil {
			return fmt.Errorf("error fetching workflow: %w", err)
		}
	}

	return refreshWorkflowToFile(cmd, *workflow, filePath, dryRun, minimal)
}

// ensureDirectoryExists checks if the directory exists and creates it if needed
func ensureDirectoryExists(cmd *cobra.Command, directory string, dryRun bool) error {
	_, err := os.Stat(directory)
	if os.IsNotExist(err) {
		if dryRun {
			cmd.Printf("Would create directory: %s\n", directory)
			return nil
		}

		if err := os.MkdirAll(directory, 0755); err != nil {
			return fmt.Errorf("error creating directory: %w", err)
		}
		cmd.Printf("Created directory: %s\n", directory)
		return nil
	}

	if err != nil {
		return fmt.Errorf("error accessing directory: %w", err)
	}

	return nil
}

// extractLocalWorkflows reads local workflow files and returns a map of workflow IDs to file paths
func extractLocalWorkflows(directory string) (map[string]string, error) {
	localFiles := make(map[string]string)

	files, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return localFiles, nil
		}
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			continue
		}

		filePath := filepath.Join(directory, file.Name())
		workflowID, err := ExtractWorkflowIDFromFile(filePath)
		if err != nil || workflowID == "" {
			continue
		}

		if existingPath, exists := localFiles[workflowID]; exists {
			existingExt := strings.ToLower(filepath.Ext(existingPath))
			currentExt := strings.ToLower(filepath.Ext(filePath))

			if (currentExt == ".yaml" || currentExt == ".yml") && existingExt == ".json" {
				localFiles[workflowID] = filePath
			}
			continue
		}

		localFiles[workflowID] = filePath
	}

	return localFiles, nil
}

// determineFilePathAndAction decides what file path and action to take for a workflow
func determineFilePathAndAction(workflow n8n.Workflow, localFiles map[string]string, directory string, output string, overwrite bool) (string, string) {
	sanitizedName := rootcmd.SanitizeFilename(workflow.Name)

	extension := ".json"

	if existingPath, exists := localFiles[*workflow.Id]; exists && output == "" {
		existingExt := strings.ToLower(filepath.Ext(existingPath))
		if existingExt == ".yaml" || existingExt == ".yml" {
			extension = existingExt
		}
	} else if strings.ToLower(output) == "yaml" || strings.ToLower(output) == "yml" {
		extension = ".yaml"
	}

	defaultPath := filepath.Join(directory, sanitizedName+extension)

	existingPath, exists := localFiles[*workflow.Id]
	if !exists || overwrite {
		return defaultPath, "Creating"
	}

	existingExt := filepath.Ext(existingPath)
	if (strings.ToLower(output) == "yaml" || strings.ToLower(output) == "yml") && strings.ToLower(existingExt) == ".json" {
		return defaultPath, "Converting"
	}

	if strings.ToLower(output) == "json" && (strings.ToLower(existingExt) == ".yaml" || strings.ToLower(existingExt) == ".yml") {
		return defaultPath, "Converting"
	}

	return existingPath, "Updating"
}

// serializeWorkflow serializes a workflow to JSON or YAML
func serializeWorkflow(workflow n8n.Workflow, filePath string, minimal bool, originalName string) ([]byte, error) {
	encoder := n8n.NewWorkflowEncoder(minimal)

	jsonData, err := encoder.EncodeToJSON(workflow)
	if err != nil {
		return nil, fmt.Errorf("error serializing workflow '%s' to JSON: %w", workflow.Name, err)
	}

	var workflowMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &workflowMap); err != nil {
		return nil, fmt.Errorf("error preparing workflow '%s' for serialization: %w", workflow.Name, err)
	}

	if originalName != "" {
		if _, exists := workflowMap["originalName"]; !exists {
			workflowMap["originalName"] = originalName
		}
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".yaml" || ext == ".yml" {
		var buf bytes.Buffer
		encoder := yaml.NewEncoder(&buf)
		encoder.SetIndent(2)

		if err := encoder.Encode(workflowMap); err != nil {
			return nil, fmt.Errorf("error serializing workflow '%s' to YAML: %w", workflow.Name, err)
		}

		if err := encoder.Close(); err != nil {
			return nil, fmt.Errorf("error finalizing YAML for workflow '%s': %w", workflow.Name, err)
		}

		return append([]byte("---\n"), buf.Bytes()...), nil
	}

	return json.MarshalIndent(workflowMap, "", "  ")
}

// workflowNeedsUpdate compares existing workflow file content with new content
func workflowNeedsUpdate(filePath string, existingPath string, content []byte, minimal bool) bool {
	if _, fileErr := os.Stat(filePath); fileErr != nil {
		return true
	}

	if !strings.EqualFold(filepath.Ext(existingPath), filepath.Ext(filePath)) {
		return true
	}

	existingContent, readErr := os.ReadFile(filePath)
	if readErr != nil {
		return true
	}

	decoder := n8n.NewWorkflowDecoder()

	existingWorkflow, err := decoder.DecodeFromBytes(existingContent)
	if err != nil {
		return true
	}

	newWorkflow, err := decoder.DecodeFromBytes(content)
	if err != nil {
		return true
	}

	return rootcmd.DetectWorkflowDrift(existingWorkflow, newWorkflow, minimal)
}

// processWorkflow handles processing of a single workflow
func processWorkflow(cmd *cobra.Command, workflow n8n.Workflow, localFiles map[string]string,
	directory string, dryRun bool, overwrite bool, output string, minimal bool) error {

	if workflow.Id == nil || *workflow.Id == "" {
		cmd.Printf("Skipping workflow '%s' with no ID\n", workflow.Name)
		return nil
	}

	filePath, action := determineFilePathAndAction(workflow, localFiles, directory, output, overwrite)
	existingPath := localFiles[*workflow.Id]

	originalName := workflow.Name
	originalNameFound := false
	if existingPath != "" {
		if value, ok := extractOriginalNameFromFile(existingPath); ok {
			originalName = value
			originalNameFound = true
		}
	}

	content, err := serializeWorkflow(workflow, filePath, minimal, originalName)
	if err != nil {
		return err
	}

	needsUpdate := true
	if action == "Updating" {
		needsUpdate = workflowNeedsUpdate(filePath, existingPath, content, minimal)
		if !needsUpdate && !originalNameFound {
			needsUpdate = true
		}
		if !needsUpdate {
			cmd.Printf("No changes for workflow '%s' (ID: %s) in file: %s\n",
				workflow.Name, *workflow.Id, filePath)
			return nil
		}
	}

	if dryRun {
		if needsUpdate || action == "Creating" || action == "Converting" {
			cmd.Printf("Would %s workflow '%s' (ID: %s) to file: %s\n",
				strings.ToLower(action), workflow.Name, *workflow.Id, filePath)
		} else {
			cmd.Printf("No changes needed for workflow '%s' (ID: %s) in file: %s\n",
				workflow.Name, *workflow.Id, filePath)
		}
		return nil
	}

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("error writing workflow '%s' to file: %w", workflow.Name, err)
	}

	cmd.Printf("%s workflow '%s' (ID: %s) to file: %s\n",
		action, workflow.Name, *workflow.Id, filePath)

	return nil
}

func refreshWorkflowToFile(cmd *cobra.Command, workflow n8n.Workflow, filePath string, dryRun bool, minimal bool) error {
	if workflow.Id == nil || *workflow.Id == "" {
		cmd.Printf("Skipping workflow '%s' with no ID\n", workflow.Name)
		return nil
	}

	originalName := workflow.Name
	originalNameFound := false
	if _, err := os.Stat(filePath); err == nil {
		if value, ok := extractOriginalNameFromFile(filePath); ok {
			originalName = value
			originalNameFound = true
		}
	}

	content, err := serializeWorkflow(workflow, filePath, minimal, originalName)
	if err != nil {
		return err
	}

	action := "Creating"
	if _, err := os.Stat(filePath); err == nil {
		action = "Updating"
	}

	needsUpdate := true
	if action == "Updating" {
		needsUpdate = workflowNeedsUpdate(filePath, filePath, content, minimal)
		if !needsUpdate && !originalNameFound {
			needsUpdate = true
		}
		if !needsUpdate {
			cmd.Printf("No changes for workflow '%s' (ID: %s) in file: %s\n",
				workflow.Name, *workflow.Id, filePath)
			return nil
		}
	}

	if dryRun {
		cmd.Printf("Would %s workflow '%s' (ID: %s) to file: %s\n",
			strings.ToLower(action), workflow.Name, *workflow.Id, filePath)
		return nil
	}

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("error writing workflow '%s' to file: %w", workflow.Name, err)
	}

	cmd.Printf("%s workflow '%s' (ID: %s) to file: %s\n",
		action, workflow.Name, *workflow.Id, filePath)

	return nil
}

func extractOriginalNameFromFile(filePath string) (string, bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	var workflowMap map[string]interface{}

	switch ext {
	case ".json":
		if err := json.Unmarshal(content, &workflowMap); err != nil {
			return "", false
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(content, &workflowMap); err != nil {
			return "", false
		}
	default:
		return "", false
	}

	if value, ok := workflowMap["originalName"]; ok {
		if name, ok := value.(string); ok && name != "" {
			return name, true
		}
	}

	return "", false
}
