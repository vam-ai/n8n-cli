// Package cmd contains commands for the n8n-cli
package cmd

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/edenreich/n8n-cli/n8n"
)

// FormatAPIBaseURL ensures the base URL ends with /api/v1
func FormatAPIBaseURL(instanceURL string) string {
	instanceURL = strings.TrimSuffix(instanceURL, "/")

	if !strings.HasSuffix(instanceURL, "/api/v1") {
		instanceURL = instanceURL + "/api/v1"
	}

	return instanceURL
}

// FindWorkflow looks up a workflow by exact name match in a list of workflows
func FindWorkflow(name string, workflows []n8n.Workflow) (string, error) {
	for _, wf := range workflows {
		if wf.Name == name {
			return *wf.Id, nil
		}
	}

	return "", fmt.Errorf("workflow with name '%s' not found", name)
}

// SanitizeFilename converts a workflow name to a valid filename
func SanitizeFilename(name string) string {
	if name == "" {
		return name
	}

	var result strings.Builder
	result.Grow(len(name))

	for _, r := range name {
		switch {
		case unicode.IsSpace(r):
			result.WriteByte('_')
		case r < 32:
			result.WriteByte('_')
		case r >= 0x1F000:
			result.WriteByte('_')
		case r == '<' || r == '>' || r == ':' || r == '"' || r == '/' || r == '\\' || r == '|' || r == '?' || r == '*':
			result.WriteByte('_')
		default:
			result.WriteRune(r)
		}
	}

	sanitized := strings.TrimRight(result.String(), " .")
	if sanitized == "" {
		return "_"
	}

	if isWindowsReservedName(sanitized) {
		sanitized = "_" + sanitized
	}

	return sanitized
}

func isWindowsReservedName(name string) bool {
	base := name
	if dot := strings.Index(base, "."); dot >= 0 {
		base = base[:dot]
	}

	base = strings.ToUpper(base)
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return true
	}

	if strings.HasPrefix(base, "COM") && len(base) == 4 {
		last := base[3]
		return last >= '1' && last <= '9'
	}

	if strings.HasPrefix(base, "LPT") && len(base) == 4 {
		last := base[3]
		return last >= '1' && last <= '9'
	}

	return false
}

// DetectWorkflowDrift compares two workflows and returns true if they differ
// This function uses reflect.DeepEqual for accurate structural comparison
// If minimal is true, both workflows will be cleaned before comparison
func DetectWorkflowDrift(actual n8n.Workflow, desired n8n.Workflow, minimal bool) bool {
	if minimal {
		actual = n8n.CleanWorkflow(actual)
		desired = n8n.CleanWorkflow(desired)
	}

	return !reflect.DeepEqual(actual, desired)
}
