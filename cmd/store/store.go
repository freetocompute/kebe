package store

import (
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var minioHostVar string
var minioSecretKeyVar string
var minioAccessKeyVar string
var databaseHostVar string
var databasePortVar int
var databaseUsernameVar string
var databasePasswordVar string
var databaseDatabaseVar string
var initializationConfigPath string

type AccountInit struct {
	Id          string `json:"id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
	Email       string `json:"email"`
}

type InitializationConfig struct {
	AuthorityId        string      `json:"authority_id"`
	RootKeyPath        string      `json:"root_key_path"`
	GenericKeyPath     string      `json:"generic_key_path"`
	RootAccountInit    AccountInit `json:"root_account_init"`
	GenericAccountInit AccountInit `json:"generic_account_init"`
}

func init() {
	Store.Flags().StringVarP(&minioHostVar, "minio-host", "m", "", "The MinIO host, like minio.awesome.com:30900")
	Store.Flags().StringVarP(&minioAccessKeyVar, "minio-access-key", "a", "", "The MinIO access key")
	Store.Flags().StringVarP(&minioSecretKeyVar, "minio-secret-key", "k", "", "The MinIO secrety key")
	Store.Flags().StringVarP(&databaseHostVar, "db-host", "s", "", "The database host, like db.awesome.com")
	Store.Flags().IntVarP(&databasePortVar, "db-port", "p", 0, "The database port, like 30032")
	Store.Flags().StringVarP(&databasePasswordVar, "db-password", "x", "", "The database password")
	Store.Flags().StringVarP(&databaseUsernameVar, "db-username", "u", "", "The database username")
	Store.Flags().StringVarP(&databaseDatabaseVar, "db-database", "d", "", "The database name")
	_ = Store.MarkFlagRequired("minio-host")
	_ = Store.MarkFlagRequired("minio-access-key")
	_ = Store.MarkFlagRequired("minio-secret-key")
	_ = Store.MarkFlagRequired("db-host")
	_ = Store.MarkFlagRequired("db-port")
	_ = Store.MarkFlagRequired("db-password")
	_ = Store.MarkFlagRequired("db-username")
	_ = Store.MarkFlagRequired("db-database")
	_ = viper.BindPFlag(configkey.MinioHost, Store.Flags().Lookup("minio-host"))
	_ = viper.BindPFlag(configkey.MinioAccessKey, Store.Flags().Lookup("minio-access-key"))
	_ = viper.BindPFlag(configkey.MinioSecretKey, Store.Flags().Lookup("minio-secret-key"))
	_ = viper.BindPFlag(configkey.DatabaseHost, Store.Flags().Lookup("db-host"))
	_ = viper.BindPFlag(configkey.DatabasePort, Store.Flags().Lookup("db-port"))
	_ = viper.BindPFlag(configkey.DatabasePassword, Store.Flags().Lookup("db-password"))
	_ = viper.BindPFlag(configkey.DatabaseUsername, Store.Flags().Lookup("db-username"))
	_ = viper.BindPFlag(configkey.DatabaseDatabase, Store.Flags().Lookup("db-database"))

	Store.AddCommand(&Initialize)
	Initialize.Flags().StringVarP(&initializationConfigPath, "initialization-config-path", "i", "", "The path the the initialization json config path")
	_ = Initialize.MarkFlagRequired("initialization-config-path")
	_ = viper.BindPFlag(configkey.StoreInitializationConfigPath, Store.Flags().Lookup("initialization-config-path"))

	Store.AddCommand(&Destroy)
	// Store.AddCommand(&RegenerateAssertions)
}

var Store = &cobra.Command{
	Use:              "store",
	Long:             "store",
	Short:            "store",
	TraverseChildren: true,
}
