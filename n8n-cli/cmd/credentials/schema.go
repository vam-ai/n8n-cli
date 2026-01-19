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
	"strings"

	rootcmd "github.com/edenreich/n8n-cli/cmd"
	"github.com/edenreich/n8n-cli/n8n"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var schemaOutput string

// SchemaCmd represents the schema command
var SchemaCmd = &cobra.Command{
	Use:   "schema CREDENTIAL_TYPE",
	Short: "Show credential data schema",
	Long:  `Show the schema for a credential type, including required fields.`,
	Args:  cobra.ExactArgs(1),
	RunE:  showCredentialSchema,
}

func init() {
	rootcmd.GetCredentialsCmd().AddCommand(SchemaCmd)

	SchemaCmd.Flags().StringVarP(&schemaOutput, "output", "o", credentialOutputJSON, "Output format: json or yaml")
}

func showCredentialSchema(cmd *cobra.Command, args []string) error {
	credentialType := args[0]

	apiKey, ok := viper.Get("api_key").(string)
	if !ok || apiKey == "" {
		return fmt.Errorf("API key not found in configuration")
	}

	instanceURL, ok := viper.Get("instance_url").(string)
	if !ok || instanceURL == "" {
		return fmt.Errorf("instance URL not found in configuration")
	}

	client := n8n.NewClient(instanceURL, apiKey)

	schema, err := client.GetCredentialSchema(credentialType)
	if err != nil {
		_, printErr := fmt.Fprintf(cmd.ErrOrStderr(), "Error fetching credential schema: %v\n", err)
		if printErr != nil {
			return fmt.Errorf("failed to write error: %v (original error: %v)", printErr, err)
		}
		return err
	}

	switch strings.ToLower(schemaOutput) {
	case credentialOutputJSON:
		jsonData, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling schema to JSON: %w", err)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(jsonData))
		return err
	case credentialOutputYAML:
		yamlData, err := yaml.Marshal(schema)
		if err != nil {
			return fmt.Errorf("error marshaling schema to YAML: %w", err)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(yamlData))
		return err
	default:
		return fmt.Errorf("unsupported output format: %s. Supported formats: json, yaml", schemaOutput)
	}
}
