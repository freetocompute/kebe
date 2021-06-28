package utils

import (
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

var password string

func init() {
	Utils.AddCommand(generatePassword)
	generatePassword.Flags().StringVarP(&password, "password", "p", "", "The password to generate a hash from")
	generatePassword.MarkFlagRequired("password")
}

var Utils = &cobra.Command{
	Use:   "utils",
	Long:  "utils",
	Short: "utils",
}

var generatePassword = &cobra.Command{
	Use:   "generate-password",
	Long:  "generate-password",
	Short: "generate-password",
	Run: func(cmd *cobra.Command, args []string) {
		hash := &Hash{}
		hashedPassword, _ := hash.Generate(password)

		fmt.Println(hashedPassword)
	},
}

// Credit: https://hackernoon.com/how-to-store-passwords-example-in-go-62712b1d2212
// https://gist.githubusercontent.com/eamonnmcevoy/c7ab5a5253712561f8dd923936646b96/raw/608ae87baf3a68053e7724dc0ab1bf10789587d4/hash.go

//Hash implements root.Hash
type Hash struct{}

//Generate a salted hash for the input string
func (c *Hash) Generate(s string) (string, error) {
	saltedBytes := []byte(s)
	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	hash := string(hashedBytes[:])
	return hash, nil
}

//Compare string to generated hash
func (c *Hash) Compare(hash string, s string) error {
	incoming := []byte(s)
	existing := []byte(hash)
	return bcrypt.CompareHashAndPassword(existing, incoming)
}