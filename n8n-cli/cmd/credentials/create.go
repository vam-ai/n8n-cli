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
package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	rootcmd "github.com/edenreich/n8n-cli/cmd"
	"github.com/edenreich/n8n-cli/n8n"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	credentialOutputText = "text"
	credentialOutputJSON = "json"
	credentialOutputYAML = "yaml"
)

var (
	credentialName     string
	credentialType     string
	credentialData     string
	credentialDataFile string
	credentialOutput   string
)

// CreateCmd represents the create credential command
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a credential",
	Long:  `Create a credential that can be used by nodes of the specified type.`,
	RunE:  createCredential,
}

func init() {
	rootcmd.GetCredentialsCmd().AddCommand(CreateCmd)

	CreateCmd.Flags().StringVarP(&credentialName, "name", "n", "", "Credential name")
	CreateCmd.Flags().StringVarP(&credentialType, "type", "t", "", "Credential type name (use credentials schema to inspect fields)")
	CreateCmd.Flags().StringVarP(&credentialData, "data", "d", "", "Credential data as a JSON object string")
	CreateCmd.Flags().StringVar(&credentialDataFile, "data-file", "", "Path to a JSON file containing credential data JSON")
	CreateCmd.Flags().StringVarP(&credentialOutput, "output", "o", credentialOutputText, "Output format: text, json, or yaml")

	_ = CreateCmd.MarkFlagRequired("name")
	_ = CreateCmd.MarkFlagRequired("type")
}

func createCredential(cmd *cobra.Command, args []string) error {
	data, err := parseCredentialData(credentialData, credentialDataFile)
	if err != nil {
		return err
	}

	apiKey, ok := viper.Get("api_key").(string)
	if !ok || apiKey == "" {
		return fmt.Errorf("API key not found in configuration")
	}

	instanceURL, ok := viper.Get("instance_url").(string)
	if !ok || instanceURL == "" {
		return fmt.Errorf("instance URL not found in configuration")
	}

	client := n8n.NewClient(instanceURL, apiKey)

	credential := n8n.Credential{
		Name: credentialName,
		Type: credentialType,
		Data: &data,
	}

	created, err := client.CreateCredential(&credential)
	if err != nil {
		_, printErr := fmt.Fprintf(cmd.ErrOrStderr(), "Error creating credential: %v\n", err)
		if printErr != nil {
			return fmt.Errorf("failed to write error: %v (original error: %v)", printErr, err)
		}
		return err
	}

	return writeCredentialOutput(cmd, created, credentialOutput)
}

func parseCredentialData(data string, dataFile string) (map[string]interface{}, error) {
	if data != "" && dataFile != "" {
		return nil, fmt.Errorf("use either --data or --data-file, not both")
	}

	var raw []byte
	var err error

	switch {
	case dataFile != "":
		raw, err = os.ReadFile(dataFile)
		if err != nil {
			return nil, fmt.Errorf("error reading credential data file: %w", err)
		}
	case data != "":
		raw = []byte(data)
	default:
		return nil, fmt.Errorf("credential data is required (use --data or --data-file)")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("invalid credential data JSON: %w", err)
	}

	if len(parsed) == 0 {
		return nil, fmt.Errorf("credential data cannot be empty")
	}

	return parsed, nil
}

func writeCredentialOutput(cmd *cobra.Command, created *n8n.CreateCredentialResponse, format string) error {
	switch strings.ToLower(format) {
	case credentialOutputJSON:
		jsonData, err := json.MarshalIndent(created, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling credential to JSON: %w", err)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(jsonData))
		return err
	case credentialOutputYAML:
		yamlData, err := yaml.Marshal(created)
		if err != nil {
			return fmt.Errorf("error marshaling credential to YAML: %w", err)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(yamlData))
		return err
	case credentialOutputText:
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Credential created successfully")
		if err != nil {
			return fmt.Errorf("failed to write output: %v", err)
		}
		if created.Id != nil {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\n", *created.Id); err != nil {
				return fmt.Errorf("failed to write credential ID: %v", err)
			}
		}
		if created.Name != "" {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", created.Name); err != nil {
				return fmt.Errorf("failed to write credential name: %v", err)
			}
		}
		if created.Type != "" {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", created.Type); err != nil {
				return fmt.Errorf("failed to write credential type: %v", err)
			}
		}
		if created.CreatedAt != nil {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Created: %s\n", created.CreatedAt.Format(time.RFC3339)); err != nil {
				return fmt.Errorf("failed to write created timestamp: %v", err)
			}
		}
		if created.UpdatedAt != nil {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Updated: %s\n", created.UpdatedAt.Format(time.RFC3339)); err != nil {
				return fmt.Errorf("failed to write updated timestamp: %v", err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s. Supported formats: text, json, yaml", format)
	}
}
