package cmd

import (
	"log"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var configAdd = &cobra.Command{
	Use:   "add",
	Short: "Adds a new configuration to the list and sets it as the active configuration",
	Run: func(cmd *cobra.Command, args []string) {

		prompt := promptui.Select{Label: "configuration type", Items: []string{authTypeOAuth, authTypeClientAuth, authTypeTokenAuth}}
		_, choice, err := prompt.Run()
		if err != nil {
			log.Fatal(err)
		}

		switch choice {
		case authTypeOAuth:
			configureOAuthCmd.Run(cmd, args)
		case authTypeClientAuth:
			configureClientAuthCmd.Run(cmd, args)
		case authTypeTokenAuth:
			configureTokenAuthCmd.Run(cmd, args)
		}
	},
}

func init() {
	ConfigCmd.AddCommand(configAdd)
}
