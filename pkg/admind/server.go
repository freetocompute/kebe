package admind

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/admind/requests"
	"github.com/freetocompute/kebe/pkg/dashboard/server"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"net/http"
)

type Server struct {
	db           *gorm.DB
	engine       *gin.Engine
	oauth2Config *oauth2.Config
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
}

func (s *Server) Run() {
	admindPort := viper.GetInt(configkey.AdminDPort)
	_ = s.engine.Run(fmt.Sprintf(":%d", admindPort))
}

func (s *Server) addAccount(c *gin.Context) {
	var addAccountReq requests.AddAccount
	json.NewDecoder(c.Request.Body).Decode(&addAccountReq)

	// TODO:: add validation
	account := models.Account{
		AccountId:   addAccountReq.AcccountId,
		DisplayName: addAccountReq.DisplayName,
		Username:    addAccountReq.Username,
		Email:       addAccountReq.Email,
	}

	s.db.Save(&account)
}

func (s *Server) verifyUser(accessToken string) (*UserInfo, error) {
	var userInfo *UserInfo
	// Create a Resty Client
	client := resty.New()
	url := config.MustGetString(configkey.OIDCProviderURL) + "/protocol/openid-connect/userinfo"
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+accessToken).
		Get(url)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == 200 {
		err = json.Unmarshal(resp.Body(), userInfo)
		if err != nil {
			return nil, err
		}

		return userInfo, err
	}

	return nil, errors.New("not found")
}

func (s *Server) addTrack(c *gin.Context) {
	var addTrackReq requests.AddTrack
	json.NewDecoder(c.Request.Body).Decode(&addTrackReq)

	logrus.Tracef("requests.AddTrack: %+v", addTrackReq)

	var snapEntry models.SnapEntry
	db := s.db.Where(&models.SnapEntry{Name: addTrackReq.SnapName}).Find(&snapEntry)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		track := models.SnapTrack{
			Name:        addTrackReq.TrackName,
			SnapEntryID: snapEntry.ID,
		}

		s.db.Save(&track)

		server.AddRisks(s.db, snapEntry.ID, track.ID)

		c.Status(http.StatusCreated)
		return
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}
