package database

import (
	"fmt"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func CreateDatabase() (*gorm.DB, error) {
	return CreateDatabaseWithDSN(getDSN())
}

func CreateDatabaseWithDSN(connectionString string) (*gorm.DB, error) {
	dsn := connectionString
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	DB = db
	return db, nil
}

func MigrateDatabase(db *gorm.DB) {
	logrus.Info("Migrating database")
	MigrateWithLog("models.Account", &models.Account{}, db)
	MigrateWithLog("models.Key", &models.Key{}, db)
	MigrateWithLog("models.SnapEntry", &models.SnapEntry{}, db)
	MigrateWithLog("models.SnapRevision", &models.SnapRevision{}, db)
}

func MigrateWithLog(name string, i interface{}, db *gorm.DB) {
	err := db.AutoMigrate(i)
	if err != nil {
		logrus.Error(err)
		panic("Failed to auto migrate: " + name)
	}
}

func CheckDBForErrorOrNoRows(db *gorm.DB) (*gorm.DB, bool) {
	if db.Error != nil {
		logrus.Error(db.Error)
		return db, false
	} else if db.RowsAffected == 0 {
		logrus.Warn("no rows found")
		return db, false
	}

	return db, true
}

func getDSN() string {
	database := viper.GetString(configkey.DatabaseDatabase)
	password := viper.GetString(configkey.DatabasePassword)
	sslMode := viper.GetString(configkey.DatabaseSSLMode)
	timezone := viper.GetString(configkey.DatabaseTimezone)
	host := viper.GetString(configkey.DatabaseHost)
	username := viper.GetString(configkey.DatabaseUsername)
	port := viper.GetInt(configkey.DatabasePort)

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		host, username, password, database, port, sslMode, timezone)

	return dsn
}
