// Package workflows contains commands for the n8n-cli workflows.
package workflows

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edenreich/n8n-cli/logger"
	"github.com/edenreich/n8n-cli/n8n"
)

func looksLikeFilePath(value string) bool {
	if value == "" {
		return false
	}

	ext := strings.ToLower(filepath.Ext(value))
	if ext == ".json" || ext == ".yaml" || ext == ".yml" {
		return true
	}

	return strings.ContainsAny(value, "/\\")
}

func validateWorkflowFileExtension(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".json" && ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("unsupported workflow file format: %s", ext)
	}
	return nil
}

func readWorkflowFromFile(filePath string) (n8n.Workflow, error) {
	var workflow n8n.Workflow

	logger.Debug("Processing file: %s", filePath)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return workflow, fmt.Errorf("error reading file: %w", err)
	}

	logger.Debug("File size: %d bytes", len(content))
	if len(content) > 0 {
		preview := string(content)
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		logger.Debug("Content preview: %s", preview)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	filename := filepath.Base(filePath)

	switch ext {
	case ".json":
		logger.Debug("Parsing as JSON: %s", filename)
		if err = json.Unmarshal(content, &workflow); err != nil {
			logger.Debug("JSON parsing error: %v", err)
			return workflow, fmt.Errorf("error parsing JSON workflow: %w", err)
		}
	case ".yaml", ".yml":
		logger.Debug("Parsing as YAML: %s", filename)

		decoder := n8n.NewWorkflowDecoder()
		workflow, err = decoder.DecodeFromYAML(content)
		if err != nil {
			logger.Debug("YAML parsing error: %v", err)
			return workflow, fmt.Errorf("error parsing YAML workflow: %w", err)
		}
	default:
		return workflow, fmt.Errorf("unsupported file format: %s", ext)
	}

	return workflow, nil
}

func findLocalWorkflowByName(directory string, name string) (string, n8n.Workflow, bool, error) {
	var emptyWorkflow n8n.Workflow
	if directory == "" || name == "" {
		return "", emptyWorkflow, false, nil
	}

	files, err := os.ReadDir(directory)
	if err != nil {
		return "", emptyWorkflow, false, fmt.Errorf("error reading directory: %w", err)
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
		if originalName, ok := extractOriginalNameFromFile(filePath); ok && originalName == name {
			workflow, err := readWorkflowFromFile(filePath)
			if err != nil {
				return "", emptyWorkflow, false, err
			}
			return filePath, workflow, true, nil
		}

		workflow, err := readWorkflowFromFile(filePath)
		if err != nil {
			return "", emptyWorkflow, false, err
		}

		if workflow.Name == name {
			return filePath, workflow, true, nil
		}
	}

	return "", emptyWorkflow, false, nil
}
