package cmd

import "github.com/spf13/cobra"

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Enables you to interact with the configuration",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(ConfigCmd)
}
