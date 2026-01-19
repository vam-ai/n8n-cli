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
	"fmt"

	rootcmd "github.com/edenreich/n8n-cli/cmd"
	"github.com/edenreich/n8n-cli/n8n"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// DeleteCmd represents the delete credential command
var DeleteCmd = &cobra.Command{
	Use:   "delete CREDENTIAL_ID",
	Short: "Delete a credential by ID",
	Long:  `Delete a credential from your n8n instance by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE:  deleteCredential,
}

func init() {
	rootcmd.GetCredentialsCmd().AddCommand(DeleteCmd)
}

func deleteCredential(cmd *cobra.Command, args []string) error {
	credentialID := args[0]

	apiKey, ok := viper.Get("api_key").(string)
	if !ok || apiKey == "" {
		return fmt.Errorf("API key not found in configuration")
	}

	instanceURL, ok := viper.Get("instance_url").(string)
	if !ok || instanceURL == "" {
		return fmt.Errorf("instance URL not found in configuration")
	}

	client := n8n.NewClient(instanceURL, apiKey)

	credential, err := client.DeleteCredential(credentialID)
	if err != nil {
		_, printErr := fmt.Fprintf(cmd.ErrOrStderr(), "Error deleting credential: %v\n", err)
		if printErr != nil {
			return fmt.Errorf("failed to write error: %v (original error: %v)", printErr, err)
		}
		return err
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Credential with ID %s has been deleted successfully\n", credentialID); err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}

	if credential != nil && credential.Name != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", credential.Name); err != nil {
			return fmt.Errorf("failed to write credential name: %v", err)
		}
	}

	return nil
}
