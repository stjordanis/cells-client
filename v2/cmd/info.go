package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var jsonFormat bool

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Displays current config",
	Long: `
Displays the current active config, show the users and the cells instance
`,
	Run: func(cmd *cobra.Command, args []string) {

		list, err := rest.GetConfigList()
		if err != nil {
			log.Fatal(err)
		}
		active := list.GetActiveConfig()

		username := active.TokenUser
		if active.TokenUser == "" {
			username = active.User
		}

		if jsonFormat {
			type info struct {
				User string `json:"user"`
				URL  string `json:"url"`
			}
			activeConfig := &info{
				User: username,
				URL:  active.Url,
			}

			data, _ := json.MarshalIndent(activeConfig, "", "\t")
			fmt.Printf("%s\n", data)
			return
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Login", "URL"})
		table.Append([]string{username, active.Url})
		table.Render()

	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
	infoCmd.Flags().BoolVarP(&jsonFormat, "json", "j", false, "returns the result as a json object")
}
