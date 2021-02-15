package cmd

import (
	"fmt"
	"log"
	"net/url"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

const authTypeTokenAuth = "token"

var configureTokenAuthCmd = &cobra.Command{
	Use: authTypeTokenAuth,
	Run: func(cm *cobra.Command, args []string) {
		var err error

		newConf := &rest.CecConfig{}

		list, err := rest.GetConfigList()
		if err != nil {
			list = &rest.ConfigList{Configs: map[string]*rest.CecConfig{}}
		}

		// interactive mode
		p := promptui.Prompt{Label: "Server Address", Validate: validUrl}
		address, err := p.Run()
		if err != nil {
			log.Fatal(err)
		}
		u, err := url.Parse(address)
		if err != nil {
			log.Fatal(err)
		}
		newConf.Url = u.String()

		p1 := promptui.Prompt{Label: "Token"}
		token, err := p1.Run()
		if err != nil {
			log.Fatal(err)
		}

		newConf.IdToken = token

		if err := list.Add("test-token", newConf); err != nil {
			log.Fatal(err)
		}

		if err := list.SaveConfigFile(); err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + err.Error())
		} else {
			fmt.Printf("%s Configuration saved under label %s\n", promptui.IconGood, color.New(color.Bold).Sprint(label))
			fmt.Printf("%s Configuration saved, you can now use the client to interact with %s.\n", promptui.IconGood, newConf.Url)
		}

	},
}

func init() {
	RootCmd.AddCommand(configureTokenAuthCmd)
}
