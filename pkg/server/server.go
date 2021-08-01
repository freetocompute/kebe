package server

import (
	"context"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/freetocompute/kebe/pkg/crypto"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"

	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/freetocompute/kebe/pkg/objectstore"
	"github.com/freetocompute/kebe/pkg/repositories"
	"github.com/freetocompute/kebe/pkg/store"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Server struct {
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

	assertsDatabase := GetDatabaseWithRootKey()

	obs := objectstore.NewObjectStore()
	bytes, _ := obs.GetFileFromBucket("root", "private-key.pem")
	rootPrivateKey, err := crypto.ParseRSAPrivateKeyFromPEM(*bytes)
	if err != nil {
		logrus.Error(err)
		panic(err)
	}

	rootAuthorityId := config.MustGetString(configkey.RootAuthority)
	signingDB := assertstest.NewSigningDB(rootAuthorityId, asserts.RSAPrivateKey(rootPrivateKey))

	bytes, _ = obs.GetFileFromBucket("generic", "private-key.pem")
	genericPrivateKey, err := crypto.ParseRSAPrivateKeyFromPEM(*bytes)
	if err != nil {
		logrus.Error(err)
		panic(err)
	}

	err = signingDB.ImportKey(asserts.RSAPrivateKey(genericPrivateKey))
	if err != nil {
		panic(err)
	}

	handler := store.NewHandler(repositories.NewAccountRepository(db), repositories.NewSnapsRepository(db))
	store := store.New(handler, assertsDatabase, rootPrivateKey, genericPrivateKey, signingDB)
	if store == nil {
		panic("store was not created, cannot continue")
	}
	store.SetupEndpoints(r)

	// Make sure all the necessary buckets exists
	err = objectstore.GetMinioClient().MakeBucket(context.Background(), "snaps", minio.MakeBucketOptions{})
	if err != nil {
		if _, ok := err.(minio.ErrorResponse); !ok {
			panic(err)
		}
	}

	err = objectstore.GetMinioClient().MakeBucket(context.Background(), "unscanned", minio.MakeBucketOptions{})
	if err != nil {
		if _, ok := err.(minio.ErrorResponse); !ok {
			panic(err)
		}
	}

	_ = r.Run()
}

var databaseCreationMutex sync.Mutex

func GetDatabaseWithRootKey() *asserts.Database {
	minioClient := objectstore.GetMinioClient()

	databaseCreationMutex.Lock()
	defer databaseCreationMutex.Unlock()

	return GetDatabaseWithRootKeyS3(minioClient)
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
