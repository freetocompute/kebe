package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/auth"
	"github.com/freetocompute/kebe/pkg/dashboard/requests"
	"github.com/freetocompute/kebe/pkg/dashboard/responses"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/macaroon.v2"
	"gorm.io/gorm"
	"io"
	"net/http"
)

type Server struct {
	engine *gin.Engine
	port int
	db *gorm.DB
}

func (s *Server) Init() {
	logrus.SetLevel(logrus.TraceLevel)
	config.LoadConfig()

	logLevelConfig := viper.GetString(configkey.LogLevel)
	l, errLevel := logrus.ParseLevel(logLevelConfig)
	if errLevel != nil {
		logrus.Error(errLevel)
	} else {
		logrus.SetLevel(l)
	}

	dashboardPort := viper.GetInt(configkey.DashboardPort)

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

	s.port = dashboardPort
	s.engine = r
}

func (s *Server) Run() {
	_ = s.engine.Run(fmt.Sprintf(":%d", s.port))
}

func (s *Server) postACL(c *gin.Context) {
	var requestACL requests.ACLRequest
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	bodyString := string(bodyBytes)
	_ = json.Unmarshal(bodyBytes, &requestACL)

	rootKeyString := config.MustGetString(configkey.MacaroonRootKey)
	rootMacaroonId := config.MustGetString(configkey.MacaroonRootId)
	rootMacaroonLocation := config.MustGetString(configkey.MacaroonRootLocation)
	m := auth.MustNewMacaroon([]byte(rootKeyString), []byte(rootMacaroonId), rootMacaroonLocation, macaroon.V1)

	dischargeKeyString := viper.GetString(configkey.MacaroonDischargeKey)
	if len(dischargeKeyString) == 0 {
		// this is panic worthy
		panic(errors.New("discharge key must be set"))
	}
	thirdPartyCaveatId := config.MustGetString(configkey.MacaroonThirdPartyCaveatId)
	thirdPartLocation := config.MustGetString(configkey.MacaroonThirdPartyLocation)
	err := m.AddThirdPartyCaveat([]byte(dischargeKeyString), []byte(thirdPartyCaveatId), thirdPartLocation)
	if err != nil {
		panic(err)
	}

	_  = m.AddFirstPartyCaveat([]byte(bodyString))

	ser, _ := auth.MacaroonSerialize(m)
	mac := &responses.Macaroon{Macaroon: ser}
	c.JSON(200, mac)
}

func (s *Server) getAccount(c *gin.Context) {
	accountUn, exists := c.Get("account")
	if exists {
		account, ok := accountUn.(*models.Account)
		if ok {
			c.JSON(http.StatusOK, &responses.AccountInfo{
				AccountId: account.AccountId,
				Snaps: map[string]map[string]map[string]string{
					"16": {},
				},
				AccountKeys: []responses.Key{},
			})
		}
	}

	c.AbortWithStatus(http.StatusUnauthorized)
}

func GetAccount(c *gin.Context) *models.Account {
	accountUn, exists := c.Get("account")
	if exists {
		account, ok := accountUn.(*models.Account)
		if ok {
			return account
		}
	}

	return nil
}

func (s *Server) registerSnapName(c *gin.Context) {
	var registerSnapName requests.RegisterSnapName
	json.NewDecoder(c.Request.Body).Decode(&registerSnapName)

	account := GetAccount(c)

	var existingSnap models.SnapEntry
	db := s.db.Where(&models.SnapEntry{Name: registerSnapName.Name}).Find(&existingSnap)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		// we found an existing snap, no luck

	} else {
		var newSnapEntry models.SnapEntry

		isDryRun := false
		dryRunString := c.Query("dry_run")
		if len(dryRunString) == 0 {
			isDryRun = false
		}

		if !isDryRun {
			snapId := uuid.New()

			newSnapEntry.SnapStoreID = snapId.String()
			newSnapEntry.Name = registerSnapName.Name
			newSnapEntry.AccountID = account.ID
			newSnapEntry.Type = "app"

			s.db.Save(&newSnapEntry)

			c.JSON(200, &responses.RegisterSnap{
				Id:  newSnapEntry.SnapStoreID,
				Name: newSnapEntry.Name,
			})
		} else {
			newSnapEntry.Name = registerSnapName.Name
			c.JSON(200, &responses.RegisterSnap{
				Name: registerSnapName.Name,
			})
		}
	}
}