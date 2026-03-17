package cmd

import (
	"github.com/PatrickMatthiesen/oh-my-dot/internal/featurecmd"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(featurecmd.NewCommand(assumedAlias, func() string {
		return viper.GetString("repo-path")
	}))
}
