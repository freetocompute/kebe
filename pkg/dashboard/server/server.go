package server

import (
	bytes2 "bytes"
	"context"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	generatedResponses "github.com/freetocompute/kebe/generated/responses"
	"github.com/freetocompute/kebe/pkg/assertions"
	"github.com/freetocompute/kebe/pkg/auth"
	dashboardRequests "github.com/freetocompute/kebe/pkg/dashboard/requests"
	dashboardResponses "github.com/freetocompute/kebe/pkg/dashboard/responses"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/freetocompute/kebe/pkg/objectstore"
	"github.com/freetocompute/kebe/pkg/sha"
	"github.com/freetocompute/kebe/pkg/snap"
	"github.com/freetocompute/kebe/pkg/store/requests"
	"github.com/freetocompute/kebe/pkg/store/responses"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/spf13/viper"
	"gopkg.in/macaroon.v2"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	var requestACL dashboardRequests.ACLRequest
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
	mac := &dashboardResponses.Macaroon{Macaroon: ser}
	c.JSON(200, mac)
}

func (s *Server) getAccount(c *gin.Context) {
	accountUn, exists := c.Get("account")
	if exists {
		account, ok := accountUn.(*models.Account)
		if ok {
			// TODO: this would actually need to be filled in
			c.JSON(http.StatusOK, &dashboardResponses.AccountInfo{
				AccountId: account.AccountId,
				Snaps: map[string]map[string]map[string]string{
					"16": {},
				},
				AccountKeys: []dashboardResponses.Key{},
			})
		}
	}
}

// TODO: find a better place for this
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

func AddRisks(db *gorm.DB, snapEntryId uint, trackId uint) {
	// TODO: fix me
	risks := []string{ "stable", "candidate", "beta", "edge" }

	// TODO: fix the need for an empty revision
	snapRevision := models.SnapRevision{
		SnapFilename:           "",
		SnapEntryID:            snapEntryId,
		SHA3_384:               "",
		Size:                   0,
	}

	db.Save(&snapRevision)

	for _, risk := range risks {
		var snapRisk models.SnapRisk
		snapRisk.SnapEntryID = snapEntryId
		snapRisk.SnapTrackID = trackId
		snapRisk.Name = risk

		snapRisk.RevisionID = snapRevision.ID

		db.Save(&snapRisk)
	}
}

func (s *Server) registerSnapName(c *gin.Context) {
	var registerSnapName dashboardRequests.RegisterSnapName
	json.NewDecoder(c.Request.Body).Decode(&registerSnapName)

	account := GetAccount(c)

	var existingSnap models.SnapEntry
	db := s.db.Where(&models.SnapEntry{Name: registerSnapName.Name}).Find(&existingSnap)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		c.AbortWithStatus(http.StatusConflict)
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

			// For now when we register a snap we are going to create the default tracks/risks
			track := models.SnapTrack{
				Name:        "latest",
				SnapEntryID: newSnapEntry.ID,
			}

			s.db.Save(&track)

			AddRisks(s.db, newSnapEntry.ID, track.ID)

			c.JSON(200, &dashboardResponses.RegisterSnap{
				Id:  newSnapEntry.SnapStoreID,
				Name: newSnapEntry.Name,
			})
		} else {
			newSnapEntry.Name = registerSnapName.Name
			c.JSON(200, &dashboardResponses.RegisterSnap{
				Name: registerSnapName.Name,
			})
		}
	}
}

func (s *Server) addAccountKey(c *gin.Context) {
	account := GetAccount(c)

	var accountKeyCreationRequest dashboardRequests.AccountKeyCreateRequest
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

		pubKey, err := assertions.GetPublicKeyFromBody(ass.Body())
		if err != nil {
			panic(err)
		}

		encodedPublicKey, err := asserts.EncodePublicKey(pubKey)
		if err != nil {
			panic(err)
		}

		pubKeyEncodedString := base64.StdEncoding.EncodeToString(encodedPublicKey)
		accountKeyToAdd := models.Key{
			Name:             ass.Name(),
			SHA3384:          ass.PublicKeyID(),
			AccountID:        account.ID,
			EncodedPublicKey: pubKeyEncodedString,
		}

		s.db.Save(&accountKeyToAdd)

		c.JSON(http.StatusOK, struct {
			PublicKey string
		} {
			PublicKey: ass.PublicKeyID(),
		})
		return
	}

	c.Data(http.StatusBadRequest, "text", []byte("Assertion type wrong, or invalid."))
}

func (s *Server) pushSnap(c *gin.Context) {
	var pushSnap requests.SnapPush
	json.NewDecoder(c.Request.Body).Decode(&pushSnap)

	if pushSnap.DryRun {
		// TODO: implement necessary checks here
		c.Status(http.StatusAccepted)
		return
	}

	// TODO: implement xdelta3 handling
	if pushSnap.DeltaFormat != "" {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var snap models.SnapEntry
	db := s.db.Where(&models.SnapEntry{Name: pushSnap.Name}).Find(&snap)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {

		rev := models.SnapRevision{
			SnapFilename:           pushSnap.UpDownId + ".snap",
			SnapEntryID:            snap.ID,
			Size:                   pushSnap.BinaryFileSize,
		}

		s.db.Save(&rev)

		// Did it have channels? if so, release it
		if len(pushSnap.Channels) > 0 {
			_ = s.releaseSnap(pushSnap.Channels, snap.ID, rev.ID)
		}

		// File saved successfully. Return proper result
		// TODO: this URL needs to be serviced by a worker thread
		c.JSON(http.StatusAccepted, &responses.Upload{
			Success:  true,
			StatusDetailsURL: config.MustGetString(configkey.DashboardURL) + "/dev/api/snap-status/" + pushSnap.UpDownId,
		})

		return
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

// The id here is the up-down id generated from the upload to /unscanned-upload/
func (s *Server) getStatus(c *gin.Context) {
	// TODO: Do whatever we need to here and then return that it's processed, some day, (hopefully!) this will need to be async!
	snapId := c.Param("id")

	// We need to move the snap from the unscanned bucket to the snaps bucket
	var rev models.SnapRevision
	snapFileName := snapId + ".snap"
	s.db.Where(&models.SnapRevision{SnapFilename: snapFileName}).Find(&rev)

	objStore := objectstore.NewObjectStore()
	objStore.Move("unscanned", "snaps", snapFileName)

	// get the sha3_384 from the file after we've moved it and update the revision
	obj, err := objectstore.GetMinioClient().GetObject(context.Background(), "snaps", snapFileName, minio.GetObjectOptions{})
	if err != nil {
		panic(err)
	}

	bytes, err := io.ReadAll(obj)

	h := crypto.SHA3_384.New()
	if err != nil {
		panic(err)
	}
	h.Write(bytes)
	actualSha3 := fmt.Sprintf("%x", h.Sum(nil))

	digest, _, err := sha.SnapFileSHA3_384FromReader(bytes2.NewReader(bytes))
	if err != nil {
		panic(err)
	}

	rev.SHA3_384 = actualSha3
	rev.SHA3384Encoded = digest

	s.db.Save(&rev)

	snapMeta, err2 := snap.GetSnapMetaFromBytes(bytes, "/tmp")
	if err2 == nil {
		logrus.Tracef("snapMeta: %+v", snapMeta)
		var snapEntry models.SnapEntry
		db := s.db.Where(&models.SnapEntry{Name: snapMeta.Name}).Find(&snapEntry)
		if _, ok :=database.CheckDBForErrorOrNoRows(db); ok {
			snapEntry.Type = snapMeta.Type
			snapEntry.Confinement = snapMeta.Confinement
			s.db.Save(&snapEntry)
		} else {
			logrus.Errorf("No rows found for: %s", snapId)
		}
	} else {
		logrus.Errorf("Unable to update snap meta: %s", err2)
	}

	c.JSON(http.StatusOK, &dashboardResponses.Status{
		Processed: true,
		Code:      "ready_to_release",
		Revision: int(rev.ID),
	})
}

func (s *Server) releaseSnap(channels []string, snapEntryId uint, revisionId uint) error {
	var trackForRelease string
	var riskForRelease string
	for _, cn := range channels {
		// It's possible this comes in the form:
		//   - single string values "edge" where the track is assumed to be "latest" there is no branch
		//   - two values "latest/edge" where the risk is proceeded by the track
		//   - three values "latest/edge/some_branch"
		parts := strings.Split(cn, "/")
		if len(parts) == 1 {
			riskForRelease = parts[0]
			trackForRelease = "latest"
		} else if len(parts) == 2 {
			trackForRelease = parts[0]
			riskForRelease = parts[1]
		} else if len(parts) == 3 {
			return errors.New("branches not supported yet")
		}

		// get all the tracks
		var track models.SnapTrack
		db := s.db.Where(&models.SnapTrack{SnapEntryID: snapEntryId, Name: trackForRelease}).Find(&track)
		if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
			// get all the risks
			var risk models.SnapRisk
			db = s.db.Where(&models.SnapRisk{SnapEntryID: snapEntryId, Name: riskForRelease, SnapTrackID: track.ID}).Find(&risk)
			if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
				var revision models.SnapRevision
				db = s.db.Where("id", revisionId).Find(&revision)
				if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
					risk.RevisionID = revision.ID
					s.db.Save(&risk)
				}
			}
		}
	}

	return nil
}

func (s *Server) snapRelease(c *gin.Context) {
	var snapRelease requests.SnapRelease
	json.NewDecoder(c.Request.Body).Decode(&snapRelease)

	var snapEntry models.SnapEntry
	db := s.db.Where(&models.SnapEntry{Name: snapRelease.Name}).Find(&snapEntry)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {

		var trackForRelease string
		var riskForRelease string
		for _, cn := range snapRelease.Channels {
			// It's possible this comes in the form:
			//   - single string values "edge" where the track is assumed to be "latest" there is no branch
			//   - two values "latest/edge" where the risk is proceeded by the track
			//   - three values "latest/edge/some_branch"
			parts := strings.Split(cn, "/")
			if len(parts) == 1 {
				riskForRelease = parts[0]
				trackForRelease = "latest"
			} else if len(parts) == 2 {
				trackForRelease = parts[0]
				riskForRelease = parts[1]
			} else if len(parts) == 3 {
				c.AbortWithError(http.StatusInternalServerError, errors.New("branches not supported yet"))
			}

			// get all the tracks
			var track models.SnapTrack
			db = s.db.Where(&models.SnapTrack{SnapEntryID: snapEntry.ID, Name: trackForRelease}).Find(&track)
			if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
				// get all the risks
				var risk models.SnapRisk
				db = s.db.Where(&models.SnapRisk{SnapEntryID: snapEntry.ID, Name: riskForRelease, SnapTrackID: track.ID}).Find(&risk)
				if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
					revisionNumber, err := strconv.Atoi(snapRelease.Revision)
					if err != nil {
						c.AbortWithError(http.StatusInternalServerError, err)
						return
					}

					var revision models.SnapRevision
					db = s.db.Where("id", revisionNumber).Find(&revision)
					if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
						risk.RevisionID = revision.ID
						s.db.Save(&risk)
					}
				} else {
					// TODO: we need to just create it if it didn't already exist
				}
			}
		}

		c.JSON(http.StatusOK, &dashboardResponses.SnapRelease{Success: true})
		return
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Server) getSnapChannelMap(c *gin.Context) {
	snapName := c.Param("snap")
	var snap models.SnapEntry
	db := s.db.Where(&models.SnapEntry{Name: snapName}).Find(&snap)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		var root generatedResponses.Root
		var channelMapItems []*generatedResponses.ChannelMapItems
		var revisions []*generatedResponses.RevisionsItems
		var channelItems []*generatedResponses.ChannelsItems
		var snapTracks []*generatedResponses.TracksItems

		logrus.Tracef("Getting tracks for: %s", snap.Name)

		var tracks []models.SnapTrack
		db := s.db.Where(&models.SnapTrack{SnapEntryID: snap.ID}).Find(&tracks)
		if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
			for _, track := range tracks {

				snapTracks = append(snapTracks, &generatedResponses.TracksItems{
					Name:           track.Name,
				})

				logrus.Tracef("Getting risks for track: %s", track.Name)

				var risks []models.SnapRisk
				db := s.db.Where(&models.SnapRisk{SnapEntryID: snap.ID, SnapTrackID: track.ID}).Find(&risks)
				if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
					for _, risk := range risks {
						var revision models.SnapRevision

						logrus.Tracef("Getting revision for risk: %s", risk.Name)

						db := s.db.Where("id", risk.RevisionID).Find(&revision)
						if _, ok := database.CheckDBForErrorOrNoRows(db); ok {

							logrus.Tracef("Got revision %d", revision.ID)

							channelMapItems = append(channelMapItems, &generatedResponses.ChannelMapItems{
								Architecture: "amd64",
								Channel:      track.Name + "/" + risk.Name,
								Revision: int(revision.ID),
								Progressive: &generatedResponses.Progressive{},
							})

							revisions = append(revisions, &generatedResponses.RevisionsItems{
								Architectures: []string{ "amd64" },
								Revision:      int(revision.ID),
								Version:       "1",
								Attributes:    &generatedResponses.Attributes{},
								Confinement:   "strict",
								Epoch:         &generatedResponses.Epoch{},
								Grade:         "stable",
								Sha3384:       revision.SHA3_384,
								Size: int(revision.Size),
							})

							channelItems = append(channelItems, &generatedResponses.ChannelsItems{
								Name:     track.Name + "/" + risk.Name,
								Risk:     risk.Name,
								Track:    track.Name,
							})
						}
					}
				}
			}
		}
		
		root.ChannelMap = channelMapItems
		root.Revisions = revisions

		root.Snap = &generatedResponses.Snap{
			Channels:     channelItems,
			Name:         snap.Name,
			Tracks: snapTracks,
		}

		c.JSON(http.StatusOK, &root)
		return
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Server) verifyACL(c *gin.Context) {
	var verify dashboardRequests.Verify
	err := json.NewDecoder(c.Request.Body).Decode(&verify)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	user, err := middleware.VerifyAndGetUser(s.db, verify.AuthData.Authorization)
	if err != nil {
		log.Fatalln(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if user != nil {
		v := dashboardResponses.Verify{
			Allowed:               true,
			DeviceRefreshRequired: false,
			RefreshRequired:       false,
			Account:               &dashboardResponses.VerifyAccount{
				Email:       user.Email,
				DisplayName: user.DisplayName,
				OpenId:      "oid1234",
				Verified:    true,
			},
			Device:                nil,
			LastAuth:              "2016-05-26T12:53:23Z",
			Permissions:           &[]string{ "package_access", "package_manage", "package_push", "package_register", "package_release", "package_update" },
			SnapIds:               nil,
			Channels:              nil,
		}

		c.JSON(http.StatusOK, &v)
		return
	}

	c.AbortWithStatus(http.StatusInternalServerError)
	return
}
