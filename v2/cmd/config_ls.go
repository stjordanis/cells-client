package cmd

import (
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var configList = &cobra.Command{
	Use:   "ls",
	Short: "Lists all the available configurations",
	Run: func(cmd *cobra.Command, args []string) {
		list, err := rest.GetConfigList()
		if err != nil {
			log.Fatal(err)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"label", "user", "URL", "type"})

		for k, v := range list.Configs {
			configType := "oauth"
			user := v.TokenUser
			// if it's a client-auth config, use User instead of TokenUser
			if v.TokenUser == "" {
				user = v.User
				configType = "client-auth"
			}
			table.Append([]string{k, user, v.Url, configType})
		}
		table.Render()

	},
}

func init() {
	ConfigCmd.AddCommand(configList)
}
