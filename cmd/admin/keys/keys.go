package keys

import (
	"fmt"
	"github.com/freetocompute/kebe/cmd/admin/config/configkey"
	"github.com/freetocompute/kebe/pkg/kebe/apiobjects"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/http"
	"net/url"
	"os"
	"path"
)

var keyName string
var userGPGPath string
var accountId string

func init() {
	Keys.AddCommand(&add)

	add.Flags().StringVarP(&keyName, "key-name", "k", "", "The name of the key to add to the account")
	add.Flags().StringVarP(&userGPGPath, "gpg-path", "g", "", "User GPG path")
	add.Flags().StringVarP(&accountId, "account-id", "a", "", "The account id")
	_ = add.MarkFlagRequired("key-name")
	_ = add.MarkFlagRequired("gpg-path")
	_ = add.MarkFlagRequired("account-id")
}

var Keys = &cobra.Command{
	Use:   "keys",
	Short: "keys",
}

var add = cobra.Command{
	Use:   "add",
	Short: "add",
	RunE: func(cmd *cobra.Command, args []string) error {
		os.Setenv("SNAP_GNUPG_HOME", userGPGPath)
		manager := asserts.NewGPGKeypairManager()

		privKey, err := manager.GetByName(keyName)
		if err != nil {
			return err
		}
		pubKey := privKey.PublicKey()

		encodedPublicKey, _ := asserts.EncodePublicKey(pubKey)
		fmt.Println(string(encodedPublicKey))

		var accountKey apiobjects.Key
		accountKey.AccountId = accountId
		accountKey.SHA3384 = pubKey.ID()
		accountKey.Name = keyName
		accountKey.EncodedPublicKey = string(encodedPublicKey)

		client := resty.New()
		request := client.R()
		request.SetBody(&accountKey)
		u, _ := url.Parse(viper.GetString(configkey.KebeAPIURL))
		u.Path = path.Join(u.Path, "kebe/v1/keys/add")
		logrus.Tracef("Using %s", u.String())
		resp, err := request.Post(u.String())

		if err != nil {
			return err
		}

		if resp.StatusCode() != http.StatusCreated {
			return err
		}

		fmt.Printf("%v\n", string(resp.Body()))

		return nil
	},
}
