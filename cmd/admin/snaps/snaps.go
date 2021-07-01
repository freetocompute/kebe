package snaps

import (
	"fmt"
	"github.com/freetocompute/kebe/pkg/kebe/apiobjects"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"net/http"
)

var snapName string
var developerId string
var snapType string

var snapId string
var snapPath string
var assertionPath string

var storeHost string

func init() {
	Snaps.AddCommand(Add)
	Add.Flags().StringVarP(&snapName, "name", "n", "", "Name of the snap to add")
	Add.Flags().StringVarP(&snapType, "type", "t", "", "Type of the snap to add (app, os, etc.)")
	Add.Flags().StringVarP(&developerId, "account-id", "a", "", "The account id of the developer of this snap")
	Add.Flags().StringVarP(&storeHost, "store-host", "s", "http://localhost:8080", "The host name of the store with port and protocol scheme (ex. http://localhost:8080)")
	_ = Add.MarkFlagRequired("name")
	_ = Add.MarkFlagRequired("account-id")
	_ = Add.MarkFlagRequired("type")

	Snaps.AddCommand(Upload)
	Upload.Flags().StringVarP(&snapPath, "snap-path", "s", "", "Path of snap to add")
	Upload.Flags().StringVarP(&snapId, "snap-id", "i", "", "The id of the snap to add")
	Upload.Flags().StringVarP(&assertionPath, "assertion-path", "a", "", "The path to the snap-build assertion")
	Upload.Flags().StringVarP(&storeHost, "store-host", "h", "http://localhost:8080", "The host name of the store with port and protocol scheme (ex. http://localhost:8080)")

	_ = Upload.MarkFlagRequired("snap-path")
	_ = Upload.MarkFlagRequired("snap-id")
	_ = Upload.MarkFlagRequired("assertion-path")
}

var Snaps = &cobra.Command{
	Use:   "snaps",
	Long:  "snaps",
	Short: "snaps",
}

var Upload = &cobra.Command{
	Use:   "upload",
	Short: "Upload a snap",
	Run: func(cmd *cobra.Command, args []string) {
		// Create a Resty Client
		client := resty.New()

		resp, err := client.R().
			SetFormData(map[string]string{
				"snapId": snapId,
			}).
			SetFiles(map[string]string{
				"snap":      snapPath,
				"assertion": assertionPath,
			}).
			Post(storeHost + "/kebe/v1/snaps/upload/" + snapId)

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(string(resp.Body()))
	},
}
var Add = &cobra.Command{
	Use:   "add",
	Short: "Adds a snap",
	Run: func(cmd *cobra.Command, args []string) {
		u := apiobjects.Snap{
			Name:        snapName,
			DeveloperID: developerId,
			Type:        snapType,
		}

		client := resty.New()
		request := client.R()
		request.SetBody(&u)
		resp, err := request.Post(storeHost + "/kebe/v1/snaps/add")

		if err != nil {
			fmt.Println(err)
			return
		}

		if resp.StatusCode() != http.StatusCreated {
			fmt.Println(resp.Error())
			return
		}

		fmt.Printf("%v\n", string(resp.Body()))
	},
}
