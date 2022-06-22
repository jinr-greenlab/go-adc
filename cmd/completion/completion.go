/*
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package completion

import (
	"github.com/spf13/cobra"
)

const (
	completionExample = `
Save shell completion to a file
# go-adc completion > $HOME/.go-adc_completions

Apply completions to the current bash instance
# source <(go-adc completion)
`
)

// NewCommand creates a cobra command object for generating bash completion script
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "completion",
		Short:   "Generate completion script for bash",
		Example: completionExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		},
	}
	return cmd
}
