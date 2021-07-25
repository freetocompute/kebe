package config

import (
	"errors"
	"fmt"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"strings"
	"sync"
)

var loadConfigMutex sync.Mutex
var configLoaded bool

var DefaultValues = map[string]interface{}{
	configkey.CanonicalSnapStoreURL: "https://api.snapcraft.io",
	configkey.DebugMode:             true,
	configkey.LogLevel:              "trace",
	configkey.RequestLogger: false,
	configkey.MinioHost:             "localhost",
	configkey.MinioSecretKey:        "password",
	configkey.MinioAccessKey:        "user",
	configkey.MinioSecure: false,
	configkey.DatabaseUsername:      "manager",
	configkey.DatabaseDatabase:      "store",
	configkey.DatabaseHost:          "localhost",
	configkey.DatabasePort:          5432,
	configkey.DatabaseSSLMode:       "disable",
	configkey.DatabaseTimezone:      "America/New_York",
	configkey.DatabasePassword:      "password",
	configkey.LoginPort: 8890,
	configkey.DashboardPort: 8891,
}

func LoadConfig() {
	loadConfigMutex.Lock()
	defer loadConfigMutex.Unlock()
	if !configLoaded {
		configLoaded = true

		explicitConfigFile := os.Getenv("CONFIG_FILE")
		if explicitConfigFile != "" {
			fmt.Printf("CONFIG_FILE: %s\n", explicitConfigFile)
			viper.SetConfigFile(explicitConfigFile)
		} else {
			viper.SetConfigName("config")
			viper.SetConfigType("yaml")
			viper.AddConfigPath("/opt/kebe-store") // path to look for the config file in

			otherPath := os.Getenv("CONFIG_FILE_PATH")
			viper.AddConfigPath(otherPath)
		}

		// set defaults first
		for key, val := range DefaultValues {
			viper.SetDefault(key, val)
		}

		viper.SetEnvPrefix("kebe")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv()

		err := viper.ReadInConfig() // Find and read the config file
		if err != nil {             // Handle errors reading the config file
			logrus.Warn("Config file not found, using defaults")
		}
	}
}

func MustGetString(key string) string{
	val := viper.GetString(key)
	if len(val) == 0 {
		panic(errors.New("failed to get " + key))
	}

	return val
}

func MustGetInt32(key string) int32 {
	if viper.IsSet(key) {
		val := viper.GetInt32(key)
		return val
	}
	panic("key not found: " + key)
}