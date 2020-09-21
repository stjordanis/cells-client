package cmd

import "github.com/spf13/cobra"

var configAdd = &cobra.Command{
	Use:   "add",
	Short: "Adds a new configuration to the list and sets it as the active configuration",
	Run:   configureOAuthCmd.Run,
}

func init() {
	ConfigCmd.AddCommand(configAdd)
}
