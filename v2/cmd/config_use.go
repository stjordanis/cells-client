package cmd

import (
	"fmt"
	"log"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var label string

var configUse = &cobra.Command{
	Use:   "use",
	Short: "Sets the active configuration",
	Long:  `This command sets the active config that will be used when interacting with a server`,
	Run: func(cmd *cobra.Command, args []string) {
		bold := color.New(color.Bold)

		list, err := rest.GetConfigList()
		if err != nil {
			log.Fatal(err)
		}

		// non interactive mode with arg
		if label != "" {
			if err := list.SetActiveConfig(label); err != nil {
				log.Fatal(err)
			}
		} else {
			// interactive mode with promptui
			var items []string
			for k, _ := range list.Configs {
				items = append(items, k)
			}

			if len(items) > 0 {
				pSelect := promptui.Select{Label: "Please select the active configuration", Items: items, Size: len(items)}
				_, result, err := pSelect.Run()
				if err != nil {
					return
				}

				if err := list.SetActiveConfig(result); err != nil {
					log.Fatal(err)
				}
			}
		}

		if err := list.SaveConfigFile(); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("The active configuration is: %s\n", bold.Sprint(list.ActiveConfig))
	},
}

func init() {
	ConfigCmd.AddCommand(configUse)
	configUse.Flags().StringVarP(&label, "label", "l", "", "non interactive way to set the active config")
}
