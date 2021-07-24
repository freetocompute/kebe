package store

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/crypto"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/freetocompute/kebe/pkg/objectstore"
	"github.com/freetocompute/kebe/pkg/store/requests"
	"github.com/freetocompute/kebe/pkg/store/responses"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"
	"github.com/snapcore/snapd/snap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
)

var databaseCreationMutex sync.Mutex

type Store struct {
	db              *gorm.DB
	assertsDatabase *asserts.Database
	rootStoreKey    *rsa.PrivateKey
	genericPrivateKey *rsa.PrivateKey
	signingDB       *assertstest.SigningDB
}

func NewStore(db *gorm.DB) *Store {
	assertsDatabase := GetDatabaseWithRootKey()

	obs := objectstore.NewObjectStore()
	bytes, _ := obs.GetFileFromBucket("root", "private-key.pem")
	rootPrivateKey, err := crypto.ParseRSAPrivateKeyFromPEM(*bytes)
	if err != nil {
		logrus.Error(err)
		return nil
	}

	rootAuthorityId := config.MustGetString(configkey.RootAuthority)
	signingDB := assertstest.NewSigningDB(rootAuthorityId, asserts.RSAPrivateKey(rootPrivateKey))

	bytes, _ = obs.GetFileFromBucket("generic", "private-key.pem")
	genericPrivateKey, err := crypto.ParseRSAPrivateKeyFromPEM(*bytes)
	if err != nil {
		logrus.Error(err)
		return nil
	}

	signingDB.ImportKey(asserts.RSAPrivateKey(genericPrivateKey))

	return &Store{
		db:              db,
		assertsDatabase: assertsDatabase,
		rootStoreKey:    rootPrivateKey,
		signingDB:       signingDB,
		genericPrivateKey: genericPrivateKey,
	}
}

func GetDatabaseWithRootKey() *asserts.Database {
	minioClient := objectstore.GetMinioClient()

	databaseCreationMutex.Lock()
	defer databaseCreationMutex.Unlock()

	var db *asserts.Database
	db = GetDatabaseWithRootKeyS3(minioClient)
	return db
}

func (s *Store) snapDownload(c *gin.Context) {
	snapFilename := c.Param("filename")
	obs := objectstore.NewObjectStore()
	bytes, _ := obs.GetFileFromBucket("snaps", snapFilename)
	c.Writer.Write(*bytes)
}

func (s *Store) getRevision(channel string, snapName string) (*models.SnapRevision, *models.SnapEntry){
	var snapEntry models.SnapEntry
	db := s.db.Where("name", snapName).Preload(clause.Associations).Find(&snapEntry)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		channelParts := strings.Split(channel, "/")
		var track string
		var risk string
		if len(channelParts) == 1 {
			if channelParts[0] == "beta" || channelParts[0] == "edge" || channelParts[0] == "stable" || channelParts[0] == "candidate" {
				track = "latest"
				risk = channelParts[0]
			} else {
				track = channelParts[0]
				risk = "stable"
			}
		} else if len(channelParts) == 2 {
			track = channelParts[0]
			risk = channelParts[1]
		} else {
			return nil, nil
		}
		var snapTrack models.SnapTrack
		var snapRisk models.SnapRisk
		//var snapRevision models.SnapRevision
		db := s.db.Where(&models.SnapTrack{SnapEntryID: snapEntry.ID, Name: track}).Find(&snapTrack)
		if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
			db := s.db.Preload(clause.Associations).Where(&models.SnapRisk{SnapEntryID: snapEntry.ID, Name: risk, SnapTrackID: snapTrack.ID}).Find(&snapRisk)
			if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
				return &snapRisk.Revision, &snapEntry
			}
		}
	}

	return nil, nil
}

func (s *Store) snapRefresh(c *gin.Context) {
	request := c.Request
	writer := c.Writer

	var actionRequest requests.SnapActionRequest
	err := json.NewDecoder(request.Body).Decode(&actionRequest)
	if err != nil {
		logrus.Error(err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writer.Header().Set("Content-Type", "application/json")

	for _, action := range actionRequest.Actions {
		var snapEntry models.SnapEntry
		db := s.db.Where("name", action.Name).Preload(clause.Associations).Preload("Revisions").Preload("Account").Find(&snapEntry)
		if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
			if action.Action == "download" {
				logrus.Infof("We know about this snap %s, its id is %s we we'll try to handle it.", snapEntry.Name, snapEntry.SnapStoreID)

				snapRevision, _ := s.getRevision(action.Channel, action.Name)
				if snapRevision != nil {
					storeSnap, err := snapEntry.ToStoreSnap(snapRevision)
					if err != nil {
						logrus.Error(err)
						c.AbortWithStatus(http.StatusBadRequest)
						return
					}
					//
					actionResult := responses.SnapActionResult{
						Result:      "download",
						InstanceKey: "download-1",
						SnapID:      snapEntry.SnapStoreID,
						Name:        snapEntry.Name,
						Snap:        storeSnap,
					}

					actionResultList := responses.SnapActionResultList{
						Results: []*responses.SnapActionResult{
							&actionResult,
						},
						ErrorList: nil,
					}

					c.JSON(http.StatusOK, &actionResultList)
					return
				}
			} else if action.Action == "install" {
				logrus.Infof("We know about this snap %s, its id is %s we we'll try to handle it.", snapEntry.Name, snapEntry.SnapStoreID)

				snapRevision, _ := s.getRevision(action.Channel, action.Name)
				if snapRevision != nil {
					storeSnap, err := snapEntry.ToStoreSnap(snapRevision)
					if err != nil {
						logrus.Error(err)
						c.AbortWithStatus(http.StatusBadRequest)
						return
					}

					storeSnap.Architectures = []string{"amd64"}
					storeSnap.Confinement = snapEntry.Confinement

					actionResult := responses.SnapActionResult{
						Result:      "install",
						InstanceKey: "install-1",
						SnapID:      snapEntry.SnapStoreID,
						Name:        snapEntry.Name,
						Snap:        storeSnap,
					}

					actionResultList := responses.SnapActionResultList{
						Results: []*responses.SnapActionResult{
							&actionResult,
						},
						ErrorList: nil,
					}

					c.JSON(http.StatusOK, &actionResultList)
					return
				}
			}
		}
	}
}

func (s *Store) getSnapSections(c *gin.Context) {
	writer := c.Writer
	logrus.Trace("/api/v1/snaps/sections")
	writer.Header().Set("Content-Type", "application/hal+json")

	sections := responses.SectionResults{
		Payload: responses.Payload{
			Sections: []responses.Section{
				{Name: "general"},
			},
		},
	}

	bytes, err := json.Marshal(&sections)
	if err != nil {
		panic(err)
	}

	writer.Write(bytes)
}

func (s *Store) findSnap(c *gin.Context) {
	var snapEntry models.SnapEntry

	searchResult := responses.SearchV2Results{
		ErrorList: nil,
	}

	// TODO: implement query parameters
	// q : search term, assume name right now
	name := c.Query("q")
	db := s.db.Preload(clause.Associations).Where(&models.SnapEntry{Name: name}).Find(&snapEntry)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		results := func() []responses.StoreSearchResult {
			var results []responses.StoreSearchResult

			snapType := snap.TypeApp
			switch snapEntry.Type {
			case "os":
				snapType = snap.TypeOS
				break
			case "snapd":
				snapType = snap.TypeSnapd
				break
			case "base":
				snapType = snap.TypeBase
				break
			case "gadget":
				snapType = snap.TypeGadget
				break
			case "kernel":
				snapType = snap.TypeKernel
			}

			results = append(results, responses.StoreSearchResult{
				Revision: responses.StoreSearchChannelSnap{
					StoreSnap: responses.StoreSnap{
						Confinement: snapEntry.Confinement,
						CreatedAt:   snapEntry.CreatedAt.String(),
						Name:        snapEntry.Name,
						// TODO: need to fix this properly
						Revision:    1,
						SnapID:      snapEntry.SnapStoreID,
						Type: snapType,
						Publisher: snap.StoreAccount{ID: snapEntry.Account.AccountId, Username: snapEntry.Account.Username, DisplayName: snapEntry.Account.DisplayName },
					},
				},
				Snap: responses.StoreSnap{
					Confinement: snapEntry.Confinement,
					CreatedAt: snapEntry.CreatedAt.String(),
					Name: snapEntry.Name,
					// TODO: need to fix this properly
					Revision:    1,
					SnapID:   snapEntry.SnapStoreID,
					Type: snapType,
					Publisher: snap.StoreAccount{ID: snapEntry.Account.AccountId, Username: snapEntry.Account.Username, DisplayName: snapEntry.Account.DisplayName },
				},
				Name:     snapEntry.Name,
				SnapID:   snapEntry.SnapStoreID,
			})

			return results
		}()

		searchResult.Results = results
	}

	logrus.Infof("%+v", searchResult)

	c.Writer.Header().Set("Content-Type", "application/json")
	bytes, _ := json.Marshal(&searchResult)
	c.Writer.Write(bytes)
}

func (s *Store) getSnapNames(c *gin.Context) {
	writer := c.Writer
	logrus.Trace("/api/v1/snaps/names")

	writer.Header().Set("Content-Type", "application/hal+json")

	var snaps []models.SnapEntry

	// TODO: would need to implement private and filter here
	s.db.Find(&snaps)
	catalogItems := responses.CatalogResults{
		Payload: responses.CatalogPayload{
			Items: []responses.CatalogItem{},
		},
	}

	for _, s := range snaps {
		catalogItems.Payload.Items = append(catalogItems.Payload.Items, responses.CatalogItem{
			Name:    s.Name,
			// TODO: implement version
			Version: "none provided",
			// TODO: implement summary
			Summary: "none provided",
			// TODO: implement aliases
			Aliases: nil,
			// TODO: implement apps
			Apps:    nil,
			// TODO: implement title
			Title:   "none provided",
		})
	}

	bytes, err := json.Marshal(&catalogItems)
	if err != nil {
		panic(err)
	}

	writer.Write(bytes)
}

func (s *Store) saveFileToTemp(c *gin.Context, snapFile *multipart.FileHeader) (string, string) {
	// Generate random file name for the new uploaded file so it doesn't override the old file with same name
	snapFileId := uuid.New().String()
	newFileName :=  snapFileId + ".snap"

	// The file is received, so let's save it
	if err := c.SaveUploadedFile(snapFile, "/tmp/"+newFileName); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "Unable to save the file",
		})
		return "", ""
	}

	return newFileName, snapFileId
}

func (s *Store) unscannedUpload(c *gin.Context) {
	snapFileData, err := c.FormFile("binary")

	// TODO: fix the actual error response to be something expected
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "No snap file is received",
		})
		return
	}

	snapFileName, id := s.saveFileToTemp(c, snapFileData)

	// TODO: create an "unscanned upload" table, store info about the upload there (like sha3-384 and base64 encoded values)

	// TODO: create "unscanned" bucket if it doesn't exist
	objStore := objectstore.NewObjectStore()
	err = objStore.SaveFileToBucket("unscanned", path.Join("/", "tmp", snapFileName))
	if err != nil {
		logrus.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, &responses.Unscanned{UploadId: id})
}

func (s *Store) authRequestIdPOST(c *gin.Context) {
	nextAuthRequest := models.AuthRequest{}
	database.DB.Save(&nextAuthRequest)

	c.JSON(http.StatusOK, requests.RequestIDResp{RequestID:strconv.Itoa(int(nextAuthRequest.ID))})
}

func (s *Store) authDevicePOST(c *gin.Context) {
	request := c.Request
	writer := c.Writer
	dec := asserts.NewDecoder(request.Body)
	for {
		got, err := dec.Decode()
		if err == io.EOF {
			break
		}
		if err != nil { // assume broken i/o
			// return nil, nil, retryErr(t, 0, "cannot read response to request for a serial: %v", err)
			panic(err)
		}
		if got.Type() == asserts.SerialRequestType {
			serialRequst := got.(*asserts.SerialRequest)

			serial := uuid.New().String()
			encodedKeyBytes, err := asserts.EncodePublicKey(serialRequst.DeviceKey())
			if err != nil {
				panic(err)
			}

			serialHeaders := map[string]interface{}{
				"brand-id":  serialRequst.BrandID(),
				"model":     serialRequst.Model(),
				"authority-id": serialRequst.BrandID(),
				"serial": serial,
				"device-key": string(encodedKeyBytes),
				"device-key-sha3-384": serialRequst.DeviceKey().ID(),
				"timestamp": "2021-03-06T15:04:00Z",
			}

			// TODO: look up actual account, get key, etc.
			ak := asserts.RSAPrivateKey(s.genericPrivateKey)
			serialAssertion, err := s.signingDB.Sign(asserts.SerialType, serialHeaders, nil, ak.PublicKey().ID())
			if err != nil {
				panic(err)
			}

			encodedSerialAssertion := asserts.Encode(serialAssertion)
			logrus.Trace("Sending serial assertion: ")
			logrus.Trace(string(encodedSerialAssertion))

			writer.Header().Set("Content-Type", asserts.MediaType)
			writer.WriteHeader(200)
			writer.Write(asserts.Encode(serialAssertion))
		}
	}
}

func (s *Store) authNonce(c *gin.Context) {
	// TODO: do we need to store this?
	nonce := responses.Nonce{Nonce: uuid.New().String()}
	c.JSON(http.StatusOK, &nonce)
}

func (s *Store) authSession(c *gin.Context) {
	c.JSON(http.StatusOK, &responses.Session{Macaroon: "12345-678-9101112"})
}

func getDatabaseConfig(minioClient *minio.Client) (*asserts.DatabaseConfig, error) {
	var trusted []asserts.Assertion
	var otherPredefined []asserts.Assertion
	buckets := []string{"root", "generic"}
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	for _, bucket := range buckets {
		objectCh := minioClient.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Recursive: true,
		})
		for object := range objectCh {
			if strings.Contains(object.Key, "assertion") {
				logrus.Tracef("Assertion key: %s", object.Key)
				filename := object.Key
				logrus.Tracef("Assertion filename: %s", filename)

				objectPtr, err := minioClient.GetObject(ctx, bucket, object.Key, minio.GetObjectOptions{})

				assertionBytes, _ := ioutil.ReadAll(objectPtr)
				logrus.Trace("assertion:")
				logrus.Trace(string(assertionBytes))
				assertion, err := asserts.Decode(assertionBytes)
				if err != nil {
					panic(err)
				} else {
					logrus.Tracef("assertion type: %s", assertion.Type())

					if assertion.Type() == asserts.AccountKeyType {
						trusted = append(trusted, assertion)
					} else if assertion.Type() == asserts.AccountType {
						trusted = append(trusted, assertion)
					} else {
						otherPredefined = append(otherPredefined, assertion)
					}
				}
			}
		}
	}

	cfg := asserts.DatabaseConfig{
		Trusted:         trusted,
		OtherPredefined: otherPredefined,
		Backstore:       asserts.NewMemoryBackstore(),
		KeypairManager:  asserts.NewMemoryKeypairManager(),
		Checkers:        nil,
	}

	return &cfg, nil
}

func GetDatabaseWithRootKeyS3(minioClient *minio.Client) *asserts.Database {
	databaseCfg, err := getDatabaseConfig(minioClient)

	db, err := asserts.OpenDatabase(databaseCfg)
	if err != nil {
		panic(err)
	}

	buckets := []string{"root", "generic"}
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	for _, bucket := range buckets {
		objectCh := minioClient.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Recursive: true,
		})
		for object := range objectCh {
			if strings.Contains(object.Key, "pem") {
				objectPtr, err := minioClient.GetObject(ctx, bucket, object.Key, minio.GetObjectOptions{})
				bytes, _ := ioutil.ReadAll(objectPtr)

				rsaPK, err := crypto.ParseRSAPrivateKeyFromPEM(bytes)
				if err != nil {
					panic(err)
				}

				assertPK := asserts.RSAPrivateKey(rsaPK)

				err = db.ImportKey(assertPK)
			}
		}
	}

	return db
}
