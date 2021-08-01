package main

import (
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/dashboard/server"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/middleware"
	"github.com/freetocompute/kebe/pkg/repositories"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cobra.OnInitialize(config.LoadConfig)
}

func main() {
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
	useRequestLogger := viper.GetBool(configkey.RequestLogger)
	db, _ := database.CreateDatabase()
	handler := server.NewDashboardHandler(repositories.NewAccountRepository(db), repositories.NewSnapsRepository(db))

	s := server.New(useRequestLogger, handler, dashboardPort)

	rootKey := config.MustGetString(configkey.MacaroonRootKey)

	checkForAuthorizedUserFunc := middleware.CheckForAuthorizedUserWithMacaroons(db, rootKey)
	s.SetupEndpoints(checkForAuthorizedUserFunc)

	s.Run()
}
