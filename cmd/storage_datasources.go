package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
	"github.com/pydio/cells-sdk-go/client/config_service"
	"github.com/pydio/cells-sdk-go/client/jobs_service"
	"github.com/pydio/cells-sdk-go/models"
)

var listDatasources = &cobra.Command{
	Use:   "list-datasources",
	Short: "ld",
	Long:  `List all the datasources`,
	Run: func(cm *cobra.Command, args []string) {

		//connects to the pydio api via the sdkConfig
		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err.Error())
		}

		/*ListDataSourcesParams contains all the parameters to send to the API endpoint
		for the list data sources operation typically these are written to a http.Request */
		params := &config_service.ListDataSourcesParams{Context: ctx}

		//assigns the datasources data retrieved above in the results variable
		result, err := apiClient.ConfigService.ListDataSources(params)
		if err != nil {
			fmt.Printf("could not list workspaces: %s\n", err.Error())
			log.Fatal(err.Error())
		}

		//prints the name of the datasources retrieved previously
		if len(result.Payload.DataSources) > 0 {
			fmt.Printf("* %d datasources	\n", len(result.Payload.DataSources))
			for _, u := range result.Payload.DataSources {
				fmt.Println("  - " + u.Name)
			}
		}

	},
}

var resyncDs = &cobra.Command{
	Use:  "resync-ds",
	Long: `Launch a resync on the specified datasource`,
	Run: func(cm *cobra.Command, args []string) {

		if len(args) != 1 {
			log.Fatal(fmt.Errorf("please provide the name of the datasource to resync"))
		}
		dsName := args[0]

		ctx, client, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err.Error())
		}

		jsonParam := fmt.Sprintf("{\"dsName\":\"%s\"}", dsName)
		body := &models.RestUserJobRequest{JobName: "datasource-resync", JSONParameters: jsonParam}

		params := &jobs_service.UserCreateJobParams{JobName: "datasource-resync", Body: body, Context: ctx}

		_, err = client.JobsService.UserCreateJob(params)
		if err != nil {
			log.Fatal(fmt.Sprintf("could not start the sync job for ds %s, cause: %s", dsName, err.Error()))
		}
	},
}

func init() {
	storageCmd.AddCommand(listDatasources)
	storageCmd.AddCommand(resyncDs)
}
