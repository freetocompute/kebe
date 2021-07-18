package middleware

import (
	"errors"
	"fmt"
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/auth"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/macaroon.v2"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

func VerifyAndGetUser(db *gorm.DB, authData string) (*models.Account, error){
	rootKey := config.MustGetString(configkey.MacaroonRootKey)

	rootS, dischargeS := GetRootMacaroonsFromString(authData)
	root, _ := auth.MacaroonDeserialize(rootS)
	discharge, _ := auth.MacaroonDeserialize(dischargeS)

	if root != nil && discharge != nil {
		err := root.Verify([]byte(rootKey), func(caveat string) error {
			return nil
		}, []*macaroon.Macaroon{discharge})

		if err != nil {
			logrus.Error(err)
			return nil, err
		} else {
			isAuthorized := false
			for _, cav := range root.Caveats() {
				if string(cav.Id) != "is-authorized-or-whatever" {
					isAuthorized = true
					break
				}
			}

			if isAuthorized {
				var email string
				for _, cav := range discharge.Caveats() {
					cavAsString := string(cav.Id)
					if strings.Contains(cavAsString, "email") {
						parts := strings.Split(cavAsString, "=")
						if len(parts) != 2 {
							return nil, errors.New("email caveat malformed")
						}
						email = parts[1]
					}
				}

				// find an account for this discharge token
				var userAccount models.Account
				db := db.Where(&models.Account{Email: email}).Find(&userAccount)
				if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
					return &userAccount, nil
				}
			}
		}
	}

	return nil, errors.New("unauthorized")
}

func CheckForAuthorizedUserWithMacaroons(db *gorm.DB, rootKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rootS, dischargeS := GetRootMacaroons(c)
		root, _ := auth.MacaroonDeserialize(rootS)
		discharge, _ := auth.MacaroonDeserialize(dischargeS)

		if root != nil && discharge != nil {
			err := root.Verify([]byte(rootKey), func(caveat string) error {
				return nil
			}, []*macaroon.Macaroon{discharge})

			if err != nil {
				logrus.Error(err)
				c.AbortWithStatus(http.StatusUnauthorized)
			} else {
				isAuthorized := false
				for _, cav := range root.Caveats() {
					if string(cav.Id) != "is-authorized-or-whatever" {
						isAuthorized = true
						break
					}
				}

				if isAuthorized {
					var email string
					for _, cav := range discharge.Caveats() {
						cavAsString := string(cav.Id)
						if strings.Contains(cavAsString, "email") {
							parts := strings.Split(cavAsString, "=")
							if len(parts) != 2 {
								c.AbortWithStatus(http.StatusUnauthorized)
							}
							email = parts[1]
							c.Set("email", email)
						} else {
							c.Set("acl", cav.Id)
						}
					}

					// find an account for this discharge token
					var userAccount models.Account
					db := db.Where(&models.Account{Email: email}).Find(&userAccount)
					if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
						c.Set("account", &userAccount)
						c.Next()
						return
					}
				}
			}
		}

		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

func GetRootMacaroons(c *gin.Context) (string, string) {
	authorizationHeaderValue := c.GetHeader("Authorization")
	tokensString := strings.TrimPrefix(authorizationHeaderValue, "Macaroon")
	tokens := strings.Split(tokensString, ",")
	var root string
	var discharge string
	for _, t := range tokens {
		fmt.Println(t)

		if strings.Contains(t, " root=") {
			root = strings.TrimPrefix(t, " root=")
		} else {
			discharge = strings.TrimPrefix(t, " discharge=")
		}
	}

	return root, discharge
}

func GetRootMacaroonsFromString(macaroonAuth string) (string, string) {
	tokensString := strings.TrimPrefix(macaroonAuth, "Macaroon")
	tokens := strings.Split(tokensString, ",")
	var root string
	var discharge string
	for _, t := range tokens {
		fmt.Println(t)

		if strings.Contains(t, " root=") {
			root = strings.TrimPrefix(t, " root=")
		} else {
			discharge = strings.TrimPrefix(t, " discharge=")
		}
	}

	return root, discharge
}