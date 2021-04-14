package store

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"github.com/freetocompute/kebe/pkg/crypto"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/freetocompute/kebe/pkg/objectstore"
	"github.com/freetocompute/kebe/pkg/store/requests"
	"github.com/freetocompute/kebe/pkg/store/responses"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"
	"github.com/snapcore/snapd/snap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

var databaseCreationMutex sync.Mutex

type Store struct {
	db              *gorm.DB
	assertsDatabase *asserts.Database
	rootStoreKey    *rsa.PrivateKey
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

	signingDB := assertstest.NewSigningDB("kebe-store", asserts.RSAPrivateKey(rootPrivateKey))

	return &Store{
		db:              db,
		assertsDatabase: assertsDatabase,
		rootStoreKey:    rootPrivateKey,
		signingDB:       signingDB,
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
				var snapEntry models.SnapEntry
				db := s.db.Where("name", action.Name).Preload(clause.Associations).Find(&snapEntry)
				if db.Error != nil {
					logrus.Error(db.Error)
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				//
				latestRevision := snapEntry.GetLatestRevision()

				storeSnap, err := snapEntry.ToStoreSnap(latestRevision.SnapFilename, 0, "")
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

			} else if action.Action == "install" {
				logrus.Infof("We know about this snap %s, its id is %s we we'll try to handle it.", snapEntry.Name, snapEntry.SnapStoreID)

				latestRevision := snapEntry.GetLatestRevision()

				storeSnap, err := snapEntry.ToStoreSnap(latestRevision.SnapFilename, 0, "")
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
	var snaps []models.SnapEntry

	s.db.Preload("Snap.Revisions").Find(&snaps)

	results := func() []responses.StoreSearchResult {
		var results []responses.StoreSearchResult
		for _, snapEntry := range snaps {

			var snapType snap.Type
			if snapEntry.Type == "app" {
				snapType = snap.TypeApp
			} else if snapEntry.Type == "os" {
				snapType = snap.TypeOS
			}

			//
			storeSearchResult := responses.StoreSearchResult{
				Revision: responses.StoreSearchChannelSnap{
					StoreSnap: responses.StoreSnap{
						Confinement: snapEntry.Confinement,
						CreatedAt: snapEntry.CreatedAt.String(),
						Name: snapEntry.Name,
						Revision: int(snapEntry.LatestRevisionID),
						SnapID:   snapEntry.SnapStoreID,
						Type: snapType,
					},
				},
				Snap: responses.StoreSnap{
					Confinement: snapEntry.Confinement,
					CreatedAt: snapEntry.CreatedAt.String(),
					Name: snapEntry.Name,
					Revision: int(snapEntry.LatestRevisionID),
					SnapID:   snapEntry.SnapStoreID,
					Type: snapType,
				},
				Name:   snapEntry.Name,
				SnapID: snapEntry.SnapStoreID,
			}

			results = append(results, storeSearchResult)
		}

		return results
	}()

	searchResult := responses.SearchV2Results{
		Results:   results,
		ErrorList: nil,
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

	catalogItems := responses.CatalogResults{
		Payload: responses.CatalogPayload{
			Items: []responses.CatalogItem{},
		},
	}

	bytes, err := json.Marshal(&catalogItems)
	if err != nil {
		panic(err)
	}

	writer.Write(bytes)
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
