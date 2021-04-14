package snaps

import (
	"fmt"
	"github.com/freetocompute/kebe/pkg/sha"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/i18n"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"time"
)

var snapPath string
var userGPGPath string
var snapStoreId string
var keyName string
var outputPath string
var developerId string

func init() {
	Snaps.AddCommand(signSnap)
	signSnap.Flags().StringVarP(&snapPath, "path", "p", "", "Snap path to sign")
	signSnap.Flags().StringVarP(&userGPGPath, "gpg-path", "g", "", "User GPG path")
	signSnap.Flags().StringVarP(&snapStoreId, "snap-store-id", "s", "", "The snap id")
	signSnap.Flags().StringVarP(&keyName, "key-name", "k", "default", "The name of the key to use")
	signSnap.Flags().StringVarP(&outputPath, "output-path", "o", "", "The output path for the sign assertion")
	signSnap.Flags().StringVarP(&developerId, "developer-id", "d", "", "The id of the developer of this snap")
	_ = signSnap.MarkFlagRequired("path")
	_ = signSnap.MarkFlagRequired("gpg-path")
	_ = signSnap.MarkFlagRequired("snap-store-id")
	_ = signSnap.MarkFlagRequired("developer-id")
}

var Snaps = &cobra.Command{
	Use:   "snaps",
	Long:  "snaps",
	Short: "snaps",
}

var signSnap = &cobra.Command{
	Use:   "sign",
	Short: "sign",
	Run: func(cmd *cobra.Command, args []string) {
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
			"snap-id":       snapStoreId,
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

		if outputPath != "" {
			err = ioutil.WriteFile(outputPath, encodedAssert, os.FileMode(0644))
			if err != nil {
				logrus.Error(err)
				return
			}
			logrus.Infof("Assertion saved to %s", outputPath)
		} else {
			_, err = fmt.Println(encodedAssert)
			if err != nil {
				logrus.Error(err)
			}
		}
		return
	},
}

var info = &cobra.Command{
	Use:   "info",
	Short: "info",
	Run: func(cmd *cobra.Command, args []string) {
		sha3_384, size, err := sha.SnapFileSHA3_384(args[0])

		if err != nil {
			panic(err)
		}

		fmt.Printf("%s, %d", sha3_384, size)
	},
}
