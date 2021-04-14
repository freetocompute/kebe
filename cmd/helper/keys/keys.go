package keys

import (
	"errors"
	"fmt"
	"github.com/freetocompute/kebe/pkg/crypto"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/i18n"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"text/tabwriter"
)

var (
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

var userGPGPath string
var keyName string

func init() {
	Keys.AddCommand(add)
	add.Flags().StringVarP(&userGPGPath, "gpg-path", "g", "", "User GPG path")
	add.Flags().StringVarP(&keyName, "key-name", "k", "default", "The name of the key to use")
	_ = add.MarkFlagRequired("gpg-path")

	Keys.AddCommand(list)
	list.Flags().StringVarP(&userGPGPath, "gpg-path", "g", "", "User GPG path")
	_ = list.MarkFlagRequired("gpg-path")
}

var Keys = &cobra.Command{
	Use:   "keys",
	Short: "keys",
}

func outputText(keys []crypto.Key) error {
	if len(keys) == 0 {
		fmt.Fprintf(Stderr, "No keys registered, see `snapcraft create-key`\n")
		return nil
	}

	w := tabWriter()
	defer w.Flush()

	fmt.Fprintln(w, i18n.G("Name\tSHA3-384"))
	for _, key := range keys {
		fmt.Fprintf(w, "%s\t%s\n", key.Name, key.Sha3_384)
	}
	return nil
}

func tabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(Stdout, 5, 3, 2, ' ', 0)
}

var list = &cobra.Command{
	Use:   "list",
	Short: "list",
	RunE: func(cmd *cobra.Command, args []string) error {
		os.Setenv("SNAP_GNUPG_HOME", userGPGPath)

		keys := []crypto.Key{}

		manager := asserts.NewGPGKeypairManager()
		collect := func(privk asserts.PrivateKey, fpr string, uid string) error {
			key := crypto.Key{
				Name:     uid,
				Sha3_384: privk.PublicKey().ID(),
			}
			keys = append(keys, key)
			return nil
		}
		err := manager.Walk(collect)
		if err != nil {
			return err
		}

		return outputText(keys)
	},
}

var add = &cobra.Command{
	Use:   "add",
	Short: "add",
	RunE: func(cmd *cobra.Command, args []string) error {
		os.Setenv("SNAP_GNUPG_HOME", userGPGPath)

		if !asserts.IsValidAccountKeyName(keyName) {
			return fmt.Errorf(i18n.G("key name %q is not valid; only ASCII letters, digits, and hyphens are allowed"), keyName)
		}

		fmt.Fprint(Stdout, i18n.G("Passphrase: "))
		passphrase, err := terminal.ReadPassword(0)
		fmt.Fprint(Stdout, "\n")
		if err != nil {
			return err
		}
		fmt.Fprint(Stdout, i18n.G("Confirm passphrase: "))
		confirmPassphrase, err := terminal.ReadPassword(0)
		fmt.Fprint(Stdout, "\n")
		if err != nil {
			return err
		}
		if string(passphrase) != string(confirmPassphrase) {
			return errors.New("passphrases do not match")
		}
		if err != nil {
			return err
		}

		manager := asserts.NewGPGKeypairManager()
		return manager.Generate(string(passphrase), keyName)
	},
}
