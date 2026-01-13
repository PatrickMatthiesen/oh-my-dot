package git

import (
	"os"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
)

// DisplaySSHAgentError displays a helpful error message when SSH agent is not configured
// exitOnError: if true, exits the program; if false, just displays a warning
func DisplaySSHAgentError(exitOnError bool) {
	fileops.ColorPrintfn(fileops.Yellow, "⚠ SSH agent not configured - cannot access remote repository.\n")
	fileops.ColorPrintln("To fix this, choose one of the following:", fileops.Reset)
	fileops.ColorPrintln("  1. Fix in this session only, run:", fileops.Reset)
	fileops.ColorPrintfn(fileops.Cyan, "     eval \"$(ssh-agent -s)\" && ssh-add\n")
	fileops.ColorPrintln("  2. Automatically add to your shell profile:", fileops.Reset)
	fileops.ColorPrintfn(fileops.Cyan, "     oh-my-dot feature add ssh-agent\n")
	fileops.ColorPrintln("  3. Manually add to your shell profile and restart:", fileops.Reset)
	fileops.ColorPrintfn(fileops.Cyan, "     eval \"$(ssh-agent -s)\" && ssh-add\n")

	if exitOnError {
		os.Exit(1)
	}
}

// CheckRemoteAccessWithHelp checks remote push permissions and provides helpful error messages
// exitOnError: if true, exits on error; if false, displays warning and continues
func CheckRemoteAccessWithHelp(exitOnError bool) {
	if err := CheckRemotePushPermission(); err != nil {
		if IsSSHAgentError(err) {
			DisplaySSHAgentError(exitOnError)
		} else {
			// Generic error message for other issues
			if exitOnError {
				fileops.ColorPrintfn(fileops.Red, "Error: %s", err)
				fileops.ColorPrintln("Cannot access remote repository. Please check your credentials and network connection.", fileops.Red)
				os.Exit(1)
			} else {
				fileops.ColorPrintfn(fileops.Yellow, "Warning: Unable to verify remote push access: %s", err)
				fileops.ColorPrintln("You may not be able to push changes to the remote repository.", fileops.Yellow)
			}
		}
	}
}
