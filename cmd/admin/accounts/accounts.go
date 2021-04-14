package accounts

import (
	"encoding/json"
	"fmt"
	"github.com/freetocompute/kebe/cmd/admin/config/configkey"
	"github.com/freetocompute/kebe/pkg/kebe/apiobjects"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
)

var jsonFileName string

func init() {
	Accounts.AddCommand(&Add)

	Add.Flags().StringVarP(&jsonFileName, "json", "j", "", "JSON file for data")
	Add.MarkFlagRequired("json")
}

var Accounts = &cobra.Command{
	Use:   "accounts",
	Long:  "accounts",
	Short: "accounts",
}

var Add = cobra.Command{
	Use:   "add",
	Short: "Adds an account",
	Run: func(cmd *cobra.Command, args []string) {
		file, _ := os.Open(jsonFileName)
		jsonbytes, _ := io.ReadAll(file)
		var account apiobjects.Account
		_ = json.Unmarshal(jsonbytes, &account)

		a := apiobjects.Account{
			DisplayName: account.DisplayName,
			Username:    account.Username,
		}

		client := resty.New()
		request := client.R()
		request.SetBody(&a)
		u, _ := url.Parse(viper.GetString(configkey.KebeAPIURL))
		u.Path = path.Join(u.Path, "kebe/v1/accounts/add")
		logrus.Tracef("Using %s", u.String())
		resp, err := request.Post(u.String())

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
