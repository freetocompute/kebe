package rollup

import (
	"encoding/json"
	"fmt"
	"github.com/freetocompute/kebe/pkg/kebe/apiobjects"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/i18n"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"
)

var snapName string
var developerId string
var snapType string
var snapPath string
var userGPGPath string
var keyName string
var storeHost string

func init() {
	Rollup.AddCommand(&snapAddSignUpload)

	snapAddSignUpload.Flags().StringVarP(&snapName, "name", "n", "", "Name of the snap to add")
	snapAddSignUpload.Flags().StringVarP(&snapType, "type", "t", "", "Type of the snap to add (app, os, etc.)")
	snapAddSignUpload.Flags().StringVarP(&developerId, "account-id", "a", "", "The account id of the developer of this snap")
	snapAddSignUpload.Flags().StringVarP(&snapPath, "path", "p", "", "Snap path to sign")
	snapAddSignUpload.Flags().StringVarP(&userGPGPath, "gpg-path", "g", "", "User GPG path")
	snapAddSignUpload.Flags().StringVarP(&keyName, "key-name", "k", "default", "The name of the key to use")
	snapAddSignUpload.Flags().StringVarP(&storeHost, "store-host", "s", "http://localhost:8080", "The host name of the store with port and protocol scheme (ex. http://localhost:8080)")
	_ = snapAddSignUpload.MarkFlagRequired("name")
	_ = snapAddSignUpload.MarkFlagRequired("account-id")
	_ = snapAddSignUpload.MarkFlagRequired("type")
	_ = snapAddSignUpload.MarkFlagRequired("path")
	_ = snapAddSignUpload.MarkFlagRequired("gpg-path")
}

var Rollup = &cobra.Command{
	Use:   "rollup",
	Long:  "rollup",
	Short: "rollup",
}

var snapAddSignUpload = cobra.Command{
	Use:   "snap-add-sign-upload",
	Short: "Add a snap, sign the snap, upload the snap",
	Run: func(cmd *cobra.Command, args []string) {
		snap := apiobjects.Snap{
			Name:        snapName,
			DeveloperID: developerId,
			Type:        snapType,
		}

		client := resty.New()
		request := client.R()
		request.SetBody(&snap)
		resp, err := request.Post(storeHost + "/kebe/v1/snaps/add")

		if err != nil {
			fmt.Println(err)
			return
		}

		if resp.StatusCode() != http.StatusCreated {
			fmt.Println(resp.Error())
			return
		}

		_ = json.Unmarshal(resp.Body(), &snap)

		snapDigest, snapSize, err := asserts.SnapFileSHA3_384(snapPath)
		if err != nil {
			logrus.Error(err)
			return
		}

		os.Setenv("SNAP_GNUPG_HOME", userGPGPath)

		gkm := asserts.NewGPGKeypairManager()
		privKey, err := gkm.GetByName(keyName)
		if err != nil {
			logrus.Errorf(i18n.G("cannot use %q key: %v"), keyName, err)
		}

		pubKey := privKey.PublicKey()
		timestamp := time.Now().Format(time.RFC3339)

		headers := map[string]interface{}{
			"developer-id":  developerId,
			"authority-id":  developerId,
			"snap-sha3-384": snapDigest,
			"snap-id":       snap.SnapStoreID,
			"snap-size":     fmt.Sprintf("%d", snapSize),
			"grade":         "stable",
			"timestamp":     timestamp,
		}

		adb, err := asserts.OpenDatabase(&asserts.DatabaseConfig{
			KeypairManager: gkm,
		})
		if err != nil {
			logrus.Errorf(i18n.G("cannot open the assertions database: %v"), err)
			return
		}

		a, err := adb.Sign(asserts.SnapBuildType, headers, nil, pubKey.ID())
		if err != nil {
			logrus.Errorf(i18n.G("cannot sign assertion: %v"), err)
			return
		}

		encodedAssert := asserts.Encode(a)

		_, snapFileName := path.Split(snapPath)
		assertionOutputPath := path.Join("/", "tmp", snapFileName+".assertion")
		err = ioutil.WriteFile(assertionOutputPath, encodedAssert, os.FileMode(0644))
		if err != nil {
			logrus.Error(err)
			return
		}
		logrus.Infof("Assertion saved to %s", assertionOutputPath)

		resp, err = client.R().
			SetFormData(map[string]string{
				"snapId": snap.SnapStoreID,
			}).
			SetFiles(map[string]string{
				"snap":      snapPath,
				"assertion": assertionOutputPath,
			}).
			Post(storeHost + "/kebe/v1/snaps/upload/" + snap.SnapStoreID)

		if err != nil {
			fmt.Println(err)
			return
		}

		defer os.Remove(assertionOutputPath)
	},
}
