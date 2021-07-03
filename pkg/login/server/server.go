package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/auth"
	"github.com/freetocompute/kebe/pkg/dashboard/responses"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/login/requests"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"gopkg.in/macaroon.v2"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
)

type Server struct {
	db *gorm.DB
}

func (s *Server) Run() {
	logrus.SetLevel(logrus.TraceLevel)
	config.LoadConfig()

	logLevelConfig := viper.GetString(configkey.LogLevel)
	l, errLevel := logrus.ParseLevel(logLevelConfig)
	if errLevel != nil {
		logrus.Error(errLevel)
	} else {
		logrus.SetLevel(l)
	}

	// Setup gin and routes
	r := gin.Default()
	if viper.GetBool(configkey.DebugMode) {
		logrus.Info("Debug mode enabled")
		r.Use(middleware.RequestLoggerMiddleware())
	} else {
		logrus.Info("Debug mode disabled")
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "KEBE STORE: PAGE_NOT_FOUND", "message": "Page not found"})
	})

	db, _ := database.CreateDatabase()
	s.db = db

	s.SetupEndpoints(r)

	loginPort := viper.GetInt(configkey.LoginPort)
	_ = r.Run(fmt.Sprintf(":%d", loginPort))
}

func MustNew(rootKey, id []byte, loc string, vers macaroon.Version) *macaroon.Macaroon {
	m, err := macaroon.New(rootKey, id, loc, vers)
	if err != nil {
		panic(err)
	}
	return m
}

func (s *Server) dischargeTokens(c *gin.Context) {
	var dischargeRequest requests.Discharge
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	_ = json.Unmarshal(bodyBytes, &dischargeRequest)

	// find an account for this discharge token
	var userAccount models.Account
	db := s.db.Where(&models.Account{Email: dischargeRequest.Email}).Find(&userAccount)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {

		clientID := viper.GetString(configkey.OIDCClientId)
		clientSecret := viper.GetString(configkey.OIDCClientSecret)
		providerURL := viper.GetString(configkey.OIDCProviderURL)

		ctx := context.Background()

		provider, err := oidc.NewProvider(ctx, providerURL)
		if err != nil {
			log.Fatal(err)
		}
		oidcConfig := &oidc.Config{
			ClientID: clientID,
		}
		verifier := provider.Verifier(oidcConfig)

		oauth2Config := oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "openid"},
		}

		oauth2Token, err := oauth2Config.PasswordCredentialsToken(ctx, userAccount.Username, dischargeRequest.Password)
		if err == nil {
			rawIDToken, ok := oauth2Token.Extra("id_token").(string)
			if ok {
				idToken, err8 := verifier.Verify(ctx, rawIDToken)
				if err8 == nil {
					resp := struct {
						OAuth2Token   *oauth2.Token
						IDTokenClaims *json.RawMessage // ID Token payload is just JSON.
					}{oauth2Token, new(json.RawMessage)}

					if err9 := idToken.Claims(&resp.IDTokenClaims); err9 == nil {
						dischargeKeyString := viper.GetString(configkey.MacaroonDischargeKey)
						if len(dischargeKeyString) == 0 {
							// this is panic worthy
							panic(errors.New("discharge key must be set"))
						}

						dm := MustNew([]byte(dischargeKeyString), []byte(dischargeRequest.CaveatId), "remote location", macaroon.LatestVersion)
						dm.AddFirstPartyCaveat([]byte("email=" + dischargeRequest.Email))

						ser, err3 := auth.MacaroonSerialize(dm)
						if err3 != nil {
							logrus.Errorf("error: %s, msg: %s", err3, "unable to serialize token")
							c.AbortWithStatus(500)
						}

						mac := &responses.DischargeMacaroon{DischargeMacaroon: ser}
						fmt.Printf("%+v\n", mac)
						bytes, _ := json.Marshal(mac)
						fmt.Println(string(bytes))
						c.JSON(200, mac)
					} else {
						c.AbortWithError(http.StatusInternalServerError, err)
					}
				} else {
					logrus.Error("Failed to verify ID Token: " + err.Error())
				}
			} else {
				logrus.Error("No id_token field in oauth2 token.")
			}
		} else {
			logrus.Error("Failed to exchange token: "+err.Error())
		}
	}

	c.AbortWithStatus(http.StatusUnauthorized)
}