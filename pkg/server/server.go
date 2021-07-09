package server

import (
	"context"
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/freetocompute/kebe/pkg/objectstore"
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

	store := store.NewStore(db)
	if store == nil {
		panic("store was not created, cannot continue")
	}
	store.SetupEndpoints(r)

	// Make sure all the necessary buckets exists
	err := objectstore.GetMinioClient().MakeBucket(context.Background(), "snaps", minio.MakeBucketOptions{})
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
