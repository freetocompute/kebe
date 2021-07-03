package helper

import (
	"fmt"
	"github.com/freetocompute/kebe/cmd/helper/keys"
	"github.com/freetocompute/kebe/cmd/helper/rollup"
	"github.com/freetocompute/kebe/cmd/helper/snaps"
	"github.com/freetocompute/kebe/cmd/helper/utils"
	"github.com/freetocompute/kebe/config"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/signtool"
	"github.com/snapcore/snapd/i18n"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"os"
)

var (
	filePath    string
	keyName     string
	userGPGPath string
	Stdin       io.Reader = os.Stdin
	Stdout      io.Writer = os.Stdout
)

func init() {
	config.LoadConfig()

	Helper.AddCommand(snaps.Snaps)
	Helper.AddCommand(rollup.Rollup)
	Helper.AddCommand(keys.Keys)
	Helper.AddCommand(utils.Utils)

	Helper.AddCommand(sign)
	sign.Flags().StringVarP(&filePath, "path", "p", "", "Path of the file to sign")
	sign.Flags().StringVarP(&keyName, "key-name", "k", "", "The name of the key to use")
	sign.Flags().StringVarP(&userGPGPath, "gpg-path", "g", "", "User GPG path")
	_ = sign.MarkFlagRequired("key-name")
	_ = sign.MarkFlagRequired("gpg-path")
}

var Helper = &cobra.Command{
	Use: "kebe-helper",
}

var sign = &cobra.Command{
	Use:   "sign",
	Short: "sign",
	RunE: func(cmd *cobra.Command, args []string) error {
		useStdin := filePath == ""

		var (
			statement []byte
			err       error
		)
		if !useStdin {
			statement, err = ioutil.ReadFile(filePath)
		} else {
			statement, err = ioutil.ReadAll(Stdin)
		}
		if err != nil {
			return fmt.Errorf(i18n.G("cannot read assertion input: %v"), err)
		}
		os.Setenv("SNAP_GNUPG_HOME", userGPGPath)
		keypairMgr := asserts.NewGPGKeypairManager()
		privKey, err := keypairMgr.GetByName(keyName)
		if err != nil {
			// TRANSLATORS: %q is the key name, %v the error message
			return fmt.Errorf(i18n.G("cannot use %q key: %v"), keyName, err)
		}

		signOpts := signtool.Options{
			KeyID:     privKey.PublicKey().ID(),
			Statement: statement,
		}

		encodedAssert, err := signtool.Sign(&signOpts, keypairMgr)
		if err != nil {
			return err
		}

		_, err = Stdout.Write(encodedAssert)
		if err != nil {
			return err
		}
		return nil
	},
}
