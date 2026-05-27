package cmd

import (
	"github.com/PatrickMatthiesen/oh-my-dot/internal/agentcmd"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(agentcmd.NewCommand(assumedAlias, func() string {
		return viper.GetString("repo-path")
	}))
}
