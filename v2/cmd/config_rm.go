package cmd

import (
	"fmt"
	"log"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var forceDelete bool

var configRemove = &cobra.Command{
	Use:   "rm",
	Short: "Removes a config form the list by its label",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		label := args[0]
		bold := color.New(color.Bold)

		list, err := rest.GetConfigList()
		if err != nil {
			log.Fatal(err)
		}

		if !forceDelete {
			pConfirm := promptui.Prompt{Label: fmt.Sprintf("You are going to remove the following configuration [%s]", bold.Sprint(label)), IsConfirm: true, Default: "Y"}
			confirm, err := pConfirm.Run()
			if err != nil || confirm == "N" {
				return
			}
		}

		if err := list.Remove(label); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("The following configuration has been removed: %s\n", bold.Sprint(label))

		var items []string
		for k, _ := range list.Configs {
			items = append(items, k)
		}

		if len(items) > 0 {
			pSelect := promptui.Select{Label: "Please set an active configuration", Items: items, Size: len(items)}
			_, result, err := pSelect.Run()
			if err != nil {
				log.Fatal(err)
			}

			if err := list.SetActiveConfig(result); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("The active configuration is %s\n", bold.Sprint(result))
		}

		// save the config once everything is done
		if err := list.SaveConfigFile(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	ConfigCmd.AddCommand(configRemove)
	configRemove.Flags().BoolVarP(&force, "force", "f", false, "Force deletion without prompt for confirmation")
}
