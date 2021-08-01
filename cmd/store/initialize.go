package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/crypto"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Destroy = cobra.Command{
	Use:   "destroy",
	Short: "Destroys the store",

	Run: func(cmd *cobra.Command, args []string) {
		minioClient := getMinioClient()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		buckets := []string{
			"root",
			"generic",
		}

		for _, bucket := range buckets {
			logrus.Infof("Remove all items in: %s", bucket)
			deleteAllItemsInBucket(minioClient, bucket)
			logrus.Infof("Removing bucket: %s", bucket)
			err := minioClient.RemoveBucket(ctx, bucket)
			if err != nil {
				logrus.Error(err)
			}
		}

		db, _ := database.CreateDatabase()
		tables := []string{
			"schema_migrations",
			"snap_branches",
			"snap_risks",
			"snap_tracks",
			"ssh_keys",
			"snap_revisions",
			"snap_entries",
			"keys",
			"accounts",
		}
		for _, t := range tables {
			db.Exec("DROP TABLE " + t)
		}

		sequences := []string{
			"accounts_id_seq",
			"keys_id_seq",
			"snap_entries_id_seq",
			"snap_revisions_id_seq",
			"ssh_keys_id_seq",
		}
		for _, s := range sequences {
			db.Exec("DROP SEQUENCE " + s)
		}
	},
}

var Initialize = cobra.Command{
	Use:   "initialize",
	Short: "Initializes the store",

	Run: func(cmd *cobra.Command, args []string) {
		// Create root key
		minioClient := getMinioClient()

		exists, err := minioClient.BucketExists(context.Background(), "root")
		if err != nil {
			panic(err)
		}

		if exists {
			fmt.Println("Bucket exists, please use destroy command if you are sure you want to start over.")
			return
		}

		exists, err = minioClient.BucketExists(context.Background(), "generic")
		if err != nil {
			panic(err)
		}

		if exists {
			fmt.Println("Bucket exists, please use destroy command if you are sure you want to start over.")
			return
		}

		var initConfig InitializationConfig
		bytes, _ := ioutil.ReadFile(initializationConfigPath)
		_ = json.Unmarshal(bytes, &initConfig)

		fmt.Printf("%+v\n", initConfig)

		makeBucketAndAddKey(minioClient, "root", initConfig.RootKeyPath, "private-key.pem")
		makeBucketAndAddKey(minioClient, "generic", initConfig.GenericKeyPath, "private-key.pem")

		// TODO: this is a redundant load
		rootKey := crypto.GetPrivateKeyFromPEMFile(initConfig.RootKeyPath)

		// create a signing database with the store's root key
		signingDB := assertstest.NewSigningDB(initConfig.AuthorityId, rootKey)
		db, _ := database.CreateDatabase()

		// generate trusted account and account key
		createTrustedAccountExt(minioClient, rootKey, rootKey.PublicKey().ID(), signingDB, initConfig.RootAccountInit.Id, initConfig.RootAccountInit.Username, "root", "default")
		rootAccount := models.Account{
			AccountId:   initConfig.RootAccountInit.Id,
			DisplayName: initConfig.RootAccountInit.DisplayName,
			Username:    initConfig.RootAccountInit.Username,
			Email:       initConfig.RootAccountInit.Email,
		}
		db.Save(&rootAccount)
		rootAccountKey := models.Key{
			Name: "default",
			//TODO: get actual sha3384, is it needed?
			SHA3384:          rootKey.PublicKey().ID(),
			EncodedPublicKey: rootKey.PublicKey().ID(),
			AccountID:        rootAccount.ID,
		}
		db.Save(&rootAccountKey)

		//
		// generate generic account, account-key and mode
		// TODO: this is a redundant load
		genericKey := crypto.GetPrivateKeyFromPEMFile(initConfig.GenericKeyPath)

		createTrustedAccountExt(minioClient, genericKey, rootKey.PublicKey().ID(), signingDB, initConfig.GenericAccountInit.Id, initConfig.GenericAccountInit.Username, "generic", "default")
		genericAccount := models.Account{
			AccountId:   initConfig.GenericAccountInit.Id,
			DisplayName: initConfig.GenericAccountInit.DisplayName,
			Username:    initConfig.GenericAccountInit.Username,
			Email:       initConfig.GenericAccountInit.Email,
		}
		db.Save(&genericAccount)
		genericAccountKey := models.Key{
			Name: "default",
			//TODO: get actual sha3384, is it needed?
			SHA3384:          genericKey.PublicKey().ID(),
			EncodedPublicKey: genericKey.PublicKey().ID(),
			AccountID:        genericAccount.ID,
		}
		db.Save(&genericAccountKey)

		fmt.Println("*******************************")
		fmt.Printf("ALL DONE. Browse to %s/%s to view your assertions.\n", viper.GetString(configkey.MinioHost), "minio/root/")
		fmt.Println("*******************************")
	},
}

func makeBucketAndAddKey(minioClient *minio.Client, bucketName string, keyPath string, keyName string) {
	// Make root bucket
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})
	for object := range objectCh {
		logrus.Tracef("object: %s", object.Key)
	}

	err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		logrus.Error(err)
	}

	bytes, _ := ioutil.ReadFile(keyPath)
	rootPrivateKey, _ := crypto.ParseRSAPrivateKeyFromPEM(bytes)
	keyString := crypto.ExportRsaPrivateKeyAsPemStr(rootPrivateKey)

	_, err = minioClient.PutObject(ctx, bucketName, keyName, strings.NewReader(keyString), int64(len(keyString)), minio.PutObjectOptions{})
	if err != nil {
		panic(err)
	}
}

func getMinioClient() *minio.Client {
	accessKey := viper.GetString(configkey.MinioAccessKey)
	secretKey := viper.GetString(configkey.MinioSecretKey)
	minioHost := viper.GetString(configkey.MinioHost)
	minioSecure := viper.GetBool(configkey.MinioSecure)

	logrus.Infof("Minio host=%s, accessKey=%s, secretKey=%s", minioHost, accessKey, secretKey)

	// Initialize minio client object.
	minioClient, err := minio.New(minioHost, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: minioSecure,
	})
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	return minioClient
}

func deleteAllItemsInBucket(minioClient *minio.Client, bucketName string) {
	objectsCh := make(chan minio.ObjectInfo)

	// Send object names that are needed to be removed to objectsCh
	go func() {
		defer close(objectsCh)
		// List all objects from a bucket-name with a matching prefix.
		for object := range minioClient.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{}) {
			if object.Err != nil {
				log.Fatalln(object.Err)
			}
			objectsCh <- object
		}
	}()

	opts := minio.RemoveObjectsOptions{
		GovernanceBypass: true,
	}

	for rErr := range minioClient.RemoveObjects(context.Background(), bucketName, objectsCh, opts) {
		fmt.Println("Error detected during deletion: ", rErr)
	}
}

func createTrustedAccountExt(minioClient *minio.Client, accountKey asserts.PrivateKey, signingKeyId string, signingDB *assertstest.SigningDB,
	accountId string, accountUsername string, bucketName string, accountKeyName string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	accountAssertion, bytes := createAccountAssertion(signingDB, signingKeyId, accountId, accountUsername)
	err := signingDB.Add(accountAssertion)
	if err != nil {
		panic(err)
	}

	_, err = minioClient.PutObject(ctx, bucketName, "account.assertion", strings.NewReader(string(bytes)), int64(len(bytes)), minio.PutObjectOptions{})
	if err != nil {
		logrus.Error(err)
	}

	_, bytes = createAccountKeyAssertion(signingDB, accountKey.PublicKey(), signingKeyId, accountAssertion, accountKeyName)
	_, err = minioClient.PutObject(ctx, bucketName, "account-key.assertion", strings.NewReader(string(bytes)), int64(len(bytes)), minio.PutObjectOptions{})
	if err != nil {
		logrus.Error(err)
	}
}

func createAccountKeyAssertion(signingDB *assertstest.SigningDB, publicKey asserts.PublicKey, keyId string, trustedAcct *asserts.Account, name string) (*asserts.AccountKey, []byte) {
	trustedAcctKeyHeaders := map[string]interface{}{
		"since":      "2015-11-20T15:04:00Z",
		"until":      "2500-11-20T15:04:00Z",
		"account-id": trustedAcct.AccountID(),
		"name":       name,
	}

	trustedAccKey := assertstest.NewAccountKey(signingDB, trustedAcct, trustedAcctKeyHeaders, publicKey, keyId)

	bytes := asserts.Encode(trustedAccKey)

	return trustedAccKey, bytes
}

func createAccountAssertion(signingDB *assertstest.SigningDB, keyId string, accountId string, storeAccountUsername string) (*asserts.Account, []byte) {
	trustedAcctHeaders := map[string]interface{}{
		"validation": "certified",
		"timestamp":  "2015-11-20T15:04:00Z",
		"account-id": accountId,
	}

	trustedAcct := assertstest.NewAccount(signingDB, storeAccountUsername, trustedAcctHeaders, keyId)

	bytes := asserts.Encode(trustedAcct)

	return trustedAcct, bytes
}
