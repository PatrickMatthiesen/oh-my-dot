package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for oh-my-dot.

The completion script needs to be sourced to provide completions in your shell.

To load completions:

Bash:
  $ source <(oh-my-dot completion bash)

  # To load completions for each session, add to your bashrc:
  $ echo 'source <(oh-my-dot completion bash)' >> ~/.bashrc

Zsh:
  $ source <(oh-my-dot completion zsh)

  # To load completions for each session, add to your zshrc:
  $ echo 'source <(oh-my-dot completion zsh)' >> ~/.zshrc

Fish:
  $ oh-my-dot completion fish | source

  # To load completions for each session:
  $ oh-my-dot completion fish > ~/.config/fish/completions/oh-my-dot.fish

PowerShell:
  PS> oh-my-dot completion powershell | Out-String | Invoke-Expression

  # To load completions for every session, add to your PowerShell profile:
  PS> oh-my-dot completion powershell >> $PROFILE
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
