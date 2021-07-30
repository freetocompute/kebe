package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/freetocompute/kebe/pkg/assertions"
	"github.com/snapcore/snapd/asserts"

	"github.com/freetocompute/kebe/pkg/dashboard/requests"

	"github.com/freetocompute/kebe/pkg/auth"
	"github.com/freetocompute/kebe/pkg/dashboard/responses"

	"github.com/freetocompute/kebe/pkg/repositories"

	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/middleware"
	storeRequests "github.com/freetocompute/kebe/pkg/store/requests"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Server struct {
	engine  *gin.Engine
	port    int
	db      *gorm.DB
	handler IDashboardHandler
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

	if viper.GetBool(configkey.RequestLogger) {
		logrus.Info("Request logger enabled")
		r.Use(middleware.RequestLoggerMiddleware())
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "KEBE STORE: PAGE_NOT_FOUND", "message": "Page not found"})
	})

	db, _ := database.CreateDatabase()
	s.db = db
	s.handler = NewDashboardHandler(repositories.NewAccountRepository(db), repositories.NewSnapsRepository(db))
	s.port = dashboardPort
	s.engine = r

	s.SetupEndpoints()
}

func (s *Server) Run() {
	_ = s.engine.Run(fmt.Sprintf(":%d", s.port))
}

func (s *Server) getAccount(c *gin.Context) {
	accountEmail := c.GetString("email")
	if accountEmail != "" {
		accountInfo, err := s.handler.GetAccount(accountEmail)
		if err == nil {
			c.JSON(http.StatusOK, &accountInfo)
			return
		}
	}
}

func (s *Server) postACL(c *gin.Context) {
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	bodyString := string(bodyBytes)

	m, err := s.handler.GetACLMacaroon(bodyString)
	if err == nil {
		ser, _ := auth.MacaroonSerialize(m)
		mac := &responses.Macaroon{Macaroon: ser}
		c.JSON(200, mac)
		return
	}

	c.Status(http.StatusInternalServerError)
}

func (s *Server) pushSnap(c *gin.Context) {
	var pushSnap storeRequests.SnapPush
	err := json.NewDecoder(c.Request.Body).Decode(&pushSnap)
	if err == nil {
		if pushSnap.DryRun {
			// TODO: implement necessary checks here
			c.Status(http.StatusAccepted)
			return
		}

		//// TODO: implement xdelta3 handling
		//// TODO: while this is not supported, the unscanned bucket could become litered with xdelta3 files
		if pushSnap.DeltaFormat != "" {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		uploadResp, err2 := s.handler.PushSnap(pushSnap.Name, pushSnap.UpDownId, uint(pushSnap.BinaryFileSize), pushSnap.Channels)
		if err2 == nil && uploadResp != nil {
			//	// File saved successfully. Return proper result
			//	// TODO: this URL needs to be serviced by a worker thread
			c.JSON(http.StatusAccepted, uploadResp)
			return
		}
	}

	logrus.Error(err)

	c.AbortWithStatus(http.StatusInternalServerError)
}

// The id here is the up-down id generated from the upload to /unscanned-upload/
func (s *Server) getStatus(c *gin.Context) {
	// TODO: Do whatever we need to here and then return that it's processed, some day, (hopefully!) this will need to be async!
	snapUpDownId := c.Param("id")

	resp, err := s.handler.GetUploadStatus(snapUpDownId)
	if err == nil && resp != nil {
		c.JSON(http.StatusOK, resp)
		return
	} else if err != nil {
		logrus.Errorf("Error getting status: %s", err)
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Server) snapRelease(c *gin.Context) {
	var rel storeRequests.SnapRelease
	err := json.NewDecoder(c.Request.Body).Decode(&rel)
	if err == nil {
		revision, err2 := strconv.Atoi(rel.Revision)
		if err2 != nil {
			logrus.Error(err2)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		released, err2 := s.handler.ReleaseSnap(rel.Name, uint(revision), rel.Channels)
		if err2 == nil {
			c.JSON(http.StatusOK, &responses.SnapRelease{Success: released})
			return
		}
	} else {
		logrus.Error(err)
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Server) getSnapChannelMap(c *gin.Context) {
	snapName := c.Param("snap")
	channelMapRoot, err := s.handler.GetSnapChannelMap(snapName)
	if err == nil && channelMapRoot != nil {
		c.JSON(http.StatusOK, channelMapRoot)
		return
	} else if err != nil {
		logrus.Error(err)
	} else {
		// logrus.Error("unknown error encountered")
		panic("unknown error encountered")
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Server) verifyACL(c *gin.Context) {
	var verify requests.Verify
	err := json.NewDecoder(c.Request.Body).Decode(&verify)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	response, err := s.handler.VerifyACL(&verify)
	if err == nil && response != nil {
		c.JSON(http.StatusOK, &response)
	} else if err != nil {
		logrus.Error(err)
	}

	c.Status(http.StatusInternalServerError)
}

func (s *Server) registerSnapName(c *gin.Context) {
	var registerSnapName requests.RegisterSnapName
	err := json.NewDecoder(c.Request.Body).Decode(&registerSnapName)
	if err == nil {
		accountEmail := c.GetString("email")

		isDryRun := false
		dryRunString := c.Query("dry_run")
		if len(dryRunString) != 0 {
			dryRun, err2 := strconv.ParseBool(dryRunString)
			if err2 == nil {
				isDryRun = dryRun
			}
		}

		resp, err2 := s.handler.RegisterSnapName(accountEmail, isDryRun, registerSnapName.Name)
		if err2 == nil && resp != nil {
			c.JSON(http.StatusOK, resp)
		}
	} else {
		logrus.Error(err)
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Server) addAccountKey(c *gin.Context) {
	accountEmail := c.GetString("email")
	if accountEmail != "" {
		var accountKeyCreationRequest requests.AccountKeyCreateRequest
		err := json.NewDecoder(c.Request.Body).Decode(&accountKeyCreationRequest)
		if err != nil {
			logrus.Error(err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if accountKeyCreationRequest.AccountKeyRequest == "" {
			c.Data(http.StatusBadRequest, "text", []byte("The account key assertion string cannot be empty"))
			return
		}

		assertion, err := asserts.Decode([]byte(accountKeyCreationRequest.AccountKeyRequest))
		if err != nil {
			c.Data(http.StatusBadRequest, "text", []byte(err.Error()))
			return
		}

		if ass, ok := assertion.(*asserts.AccountKeyRequest); ok {
			logrus.Infof("Account key public key id: %s", ass.PublicKeyID())

			pubKey, err2 := assertions.GetPublicKeyFromBody(ass.Body())
			if err2 != nil {
				panic(err2)
			}

			encodedPublicKey, err2 := asserts.EncodePublicKey(pubKey)
			if err2 != nil {
				panic(err2)
			}

			pubKeyEncodedString := base64.StdEncoding.EncodeToString(encodedPublicKey)

			acct, err2 := s.handler.AddAccountKey(accountEmail, ass.Name(), ass.PublicKeyID(), pubKeyEncodedString)
			if err2 == nil && acct != nil {
				c.JSON(http.StatusOK, struct {
					PublicKey string
				}{
					PublicKey: ass.PublicKeyID(),
				})
				return
			} else if err2 != nil {
				logrus.Error(err2)
			}

			c.Status(http.StatusInternalServerError)
		}
	}

	c.Data(http.StatusBadRequest, "text", []byte("Assertion type wrong, or invalid."))
}
