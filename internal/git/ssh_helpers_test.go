package git

import (
	"strings"
	"testing"
)

func TestSSHAgentHelp(t *testing.T) {
	tests := []struct {
		platform string
		want     []string
		absent   string
	}{
		{"windows", []string{"Get-Service ssh-agent", "Start-Service ssh-agent", "StartupType Manual", "administrator PowerShell", "OpenSSH.Client", "ssh-add"}, "eval"},
		{"linux", []string{"bash or zsh", "ssh-agent -s", "ssh-add", "OpenSSH client package"}, "Set-Service"},
		{"darwin", []string{"ssh-agent -s", "ssh-add"}, "Set-Service"},
	}
	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			help := sshAgentHelp(tt.platform)
			for _, want := range tt.want {
				if !strings.Contains(help, want) {
					t.Errorf("missing %q in guidance", want)
				}
			}
			if strings.Contains(help, tt.absent) {
				t.Errorf("guidance contains inappropriate %q", tt.absent)
			}
		})
	}
}
