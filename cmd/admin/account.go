package admin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/admind"
	"github.com/freetocompute/kebe/pkg/admind/requests"
	resty "github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var username string
var accountId string
var email string
var displayName string

const (
	LoginConfigFilename = ".loginconfig"
)

func init() {
	account.AddCommand(add)

	add.Flags().StringVarP(&username, "username", "u", "", "The username of the account")
	add.Flags().StringVarP(&accountId, "account-id", "a", "", "The account id of the user")
	add.Flags().StringVarP(&email, "email", "e", "", "The email of the user")
	add.Flags().StringVarP(&displayName, "display-name", "d", "", "The display name for the account")
	_ = add.MarkFlagRequired("email")
	_ = add.MarkFlagRequired("display-name")
	_ = add.MarkFlagRequired("account-id")
	_ = add.MarkFlagRequired("username")
}

var account = &cobra.Command{
	Use:   "account",
	Short: "account",
}

func refreshToken(refreshToken string) (*admind.LoginInfo, error) {
	client := resty.New()
	url := config.MustGetString(configkey.OIDCProviderURL) + "/protocol/openid-connect/token"
	clientId := config.MustGetString(configkey.OIDCClientId)
	clientSecret := config.MustGetString(configkey.OIDCClientSecret)
	resp, err := client.R().
		SetFormData(map[string]string{
			"grant_type":    "refresh_token",
			"client_id":     clientId,
			"refresh_token": refreshToken,
			"client_secret": clientSecret,
		}).
		Post(url)

	if err != nil {
		panic(err)
	}

	var loginInfo admind.LoginInfo
	bytes, _ := ioutil.ReadFile(LoginConfigFilename)
	err = json.Unmarshal(bytes, &loginInfo)
	if err != nil {
		panic(err)
	}

	// We we read the token from the token URL it's unmodified by the oauth2 client
	var token admind.Token
	err2 := json.Unmarshal(resp.Body(), &token)
	if err2 != nil {
		return nil, err2
	}

	expiresIn := time.Duration(token.ExpiresIn) * time.Second
	loginInfo.Token = oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       time.Now().Add(expiresIn),
	}

	bytes, err = json.Marshal(loginInfo)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(LoginConfigFilename, bytes, 0600)
	if err != nil {
		return nil, err
	}

	return &loginInfo, nil
}

var add = &cobra.Command{
	Use:   "add",
	Short: "add",
	Run: func(cmd *cobra.Command, args []string) {
		var loginInfo admind.LoginInfo
		bytes, _ := ioutil.ReadFile(LoginConfigFilename)
		err := json.Unmarshal(bytes, &loginInfo)
		if err != nil {
			panic(err)
		}

		// we've expired and we need to refresh, so for now
		// force a login
		if time.Now().After(loginInfo.Token.Expiry) {
			fmt.Printf("Token has expired, refreshing.")
			loginInfoPtr, err := refreshToken(loginInfo.Token.RefreshToken)
			if err != nil {
				panic(err)
			}

			if loginInfoPtr != nil {
				loginInfo = *loginInfoPtr
			}
		}

		addAccountRequest := requests.AddAccount{
			Username:    username,
			AcccountId:  accountId,
			Email:       email,
			DisplayName: displayName,
		}

		// Create a Resty Client
		client := resty.New()

		admindURL := config.MustGetString(configkey.AdminDURL)

		accountURL := admindURL + "/v1/admin/account"

		bytes, _ = json.Marshal(&addAccountRequest)
		resp, err := client.R().
			SetBody(bytes).
			SetHeader("Authorization", loginInfo.Token.AccessToken).
			Post(accountURL)

		if err != nil {
			panic(err)
		}

		if resp.StatusCode() != 200 {
			if resp.Error() != nil {
				panic(resp.Error())
			}
			panic("there was a problem: " + strconv.Itoa(resp.StatusCode()))
		}
	},
}
