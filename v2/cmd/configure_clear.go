package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var noKeyringDefined bool
var forceClear bool

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear current configuration",
	Long:  "Clear current authentication data from your local keyring",
	Run: func(cmd *cobra.Command, args []string) {

		list, err := rest.GetConfigList()
		if err != nil {
			log.Fatal(err)
		}

		if !forceClear {
			pConfirm := promptui.Prompt{Label: "You are going to clear everything", IsConfirm: true, Default: "Y"}
			confirm, err := pConfirm.Run()
			if err != nil || confirm == "N" {
				fmt.Println("Nothing has been removed")
				return
			}
		}

		for _, v := range list.Configs {
			if err := rest.ClearKeyring(v); err != nil {
				log.Printf("%s\n", err)
			}
		}
		fmt.Println(promptui.IconGood + " Successfully cleared the keyring")

		if err := os.Remove(rest.GetConfigFilePath()); err != nil {
			log.Fatal(err)
		}

		fmt.Println(promptui.IconGood + " Successfully removed config file")
	},
}

func init() {
	flags := clearCmd.PersistentFlags()
	helpMsg := "Explicitly tell the tool to *NOT* try to use a keyring. Only use this flag if you really know what your are doing: some sensitive information will end up stored on your file system in clear text."
	flags.BoolVar(&noKeyringDefined, "no-keyring", false, helpMsg)
	flags.BoolVarP(&forceClear, "force", "f", false, "Force deletes without confirmation")
	RootCmd.AddCommand(clearCmd)
}
