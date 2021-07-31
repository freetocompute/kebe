package store

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/freetocompute/kebe/pkg/repositories"

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
	minio "github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"
	"gorm.io/gorm"
)

var databaseCreationMutex sync.Mutex

type Store struct {
	db                *gorm.DB
	assertsDatabase   *asserts.Database
	rootStoreKey      *rsa.PrivateKey
	genericPrivateKey *rsa.PrivateKey
	signingDB         *assertstest.SigningDB
	handler           IStoreHandler
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

	err = signingDB.ImportKey(asserts.RSAPrivateKey(genericPrivateKey))
	if err != nil {
		panic(err)
	}

	handler := NewHandler(repositories.NewAccountRepository(db), repositories.NewSnapsRepository(db))

	return &Store{
		db:                db,
		assertsDatabase:   assertsDatabase,
		rootStoreKey:      rootPrivateKey,
		signingDB:         signingDB,
		genericPrivateKey: genericPrivateKey,
		handler:           handler,
	}
}

func GetDatabaseWithRootKey() *asserts.Database {
	minioClient := objectstore.GetMinioClient()

	databaseCreationMutex.Lock()
	defer databaseCreationMutex.Unlock()

	return GetDatabaseWithRootKeyS3(minioClient)
}

func (s *Store) snapDownload(c *gin.Context) {
	snapFilename := c.Param("filename")
	obs := objectstore.NewObjectStore()
	bytes, _ := obs.GetFileFromBucket("snaps", snapFilename)
	_, err := c.Writer.Write(*bytes)
	if err != nil {
		logrus.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
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

	snapActionResultList, err := s.handler.SnapRefresh(&actionRequest.Actions)
	if err == nil && snapActionResultList != nil {
		c.JSON(http.StatusOK, &snapActionResultList)
		return
	} else if err != nil {
		logrus.Error(err)
	} else {
		logrus.Error("unknown error encountered in snapRefresh")
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) getSnapSections(c *gin.Context) {
	writer := c.Writer
	logrus.Trace("/api/v1/snaps/sections")
	writer.Header().Set("Content-Type", "application/hal+json")

	result, err := s.handler.GetSections()
	if err == nil && result != nil {
		c.JSON(http.StatusOK, result)
	} else if err != nil {
		logrus.Error(err)
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) findSnap(c *gin.Context) {
	// TODO: implement query parameters
	// q : search term, assume name right now
	name := c.Query("q")
	searchResults, err := s.handler.FindSnap(name)
	if err == nil && searchResults != nil {
		logrus.Infof("%+v", searchResults)

		c.Writer.Header().Set("Content-Type", "application/json")
		bytes, _ := json.Marshal(&searchResults)
		_, err2 := c.Writer.Write(bytes)
		if err2 != nil {
			logrus.Error(err2)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		return
	} else if err != nil {
		logrus.Error(err)
	} else {
		logrus.Error("unknown error encountered handling /v2/snaps/find in findSnap")
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) getSnapNames(c *gin.Context) {
	writer := c.Writer
	logrus.Trace("/api/v1/snaps/names")

	writer.Header().Set("Content-Type", "application/hal+json")

	catalogItems, err := s.handler.GetSnapNames()
	if err == nil && catalogItems != nil {
		bytes, err := json.Marshal(catalogItems)
		if err == nil {
			_, err2 := writer.Write(bytes)
			if err2 == nil {
				return
			}
		}
	}

	if err != nil {
		logrus.Error(err)
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) saveFileToTemp(c *gin.Context, snapFile *multipart.FileHeader) (string, string) {
	// Generate random file name for the new uploaded file so it doesn't override the old file with same name
	snapFileId := uuid.New().String()
	newFileName := snapFileId + ".snap"

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

	c.JSON(http.StatusOK, requests.RequestIDResp{RequestID: strconv.Itoa(int(nextAuthRequest.ID))})
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
				"brand-id":            serialRequst.BrandID(),
				"model":               serialRequst.Model(),
				"authority-id":        serialRequst.BrandID(),
				"serial":              serial,
				"device-key":          string(encodedKeyBytes),
				"device-key-sha3-384": serialRequst.DeviceKey().ID(),
				// TODO: fix this static timestamp
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
			_, err2 := writer.Write(asserts.Encode(serialAssertion))
			if err2 == nil {
				return
			}

			logrus.Error(err2)
		}
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) authNonce(c *gin.Context) {
	// TODO: do we need to store this?
	nonce := responses.Nonce{Nonce: uuid.New().String()}
	c.JSON(http.StatusOK, &nonce)
}

func (s *Store) authSession(c *gin.Context) {
	// TODO: implement actual sessions?
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
				if err != nil {
					panic(err)
				}

				assertionBytes, _ := ioutil.ReadAll(objectPtr)
				logrus.Trace("assertion:")
				logrus.Trace(string(assertionBytes))
				assertion, err := asserts.Decode(assertionBytes)
				if err != nil {
					panic(err)
				} else {
					logrus.Tracef("assertion type: %s", assertion.Type().Name)

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
	if err != nil {
		panic(err)
	}

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
				objectPtr, err2 := minioClient.GetObject(ctx, bucket, object.Key, minio.GetObjectOptions{})
				if err2 != nil {
					panic(err2)
				}
				bytes, _ := ioutil.ReadAll(objectPtr)

				rsaPK, err2 := crypto.ParseRSAPrivateKeyFromPEM(bytes)
				if err2 != nil {
					panic(err)
				}

				assertPK := asserts.RSAPrivateKey(rsaPK)

				err2 = db.ImportKey(assertPK)
				if err2 != nil {
					panic(err)
				}
			}
		}
	}

	return db
}
