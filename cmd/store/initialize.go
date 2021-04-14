package store

import (
	"context"
	"fmt"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/crypto"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"strings"
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
	},
}

var RegenerateAssertions = cobra.Command{
	Use:   "regenerate-assertions",
	Short: "Regenerates the assertions using the existing keys",
	Run: func(cmd *cobra.Command, args []string) {
		minioClient := getMinioClient()
		if minioClient == nil {
			panic("no minio connection")
		}

		// regenerate account and account key for the root
		// regenerateRootAccountAsertions()

		// regenerate generic account, account-key and model
		_ = regenerateGenericAccountAssertions()
	},
}

func regenerateGenericAccountAssertions() error {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	minioClient := getMinioClient()
	objectPtr, err := minioClient.GetObject(ctx, "root",
		"private-key.pem", minio.GetObjectOptions{})

	if err != nil {
		logrus.Error(err)
		return err
	}

	bytes, err2 := ioutil.ReadAll(objectPtr)
	if err2 != nil {
		logrus.Error(err2)
		return err2
	}

	rootPrivateKey, err := crypto.ParseRSAPrivateKeyFromPEM(bytes)
	if err != nil {
		logrus.Error(err)
		return err
	}

	objectPtr, err = minioClient.GetObject(ctx, "generic",
		"private-key.pem", minio.GetObjectOptions{})

	if err != nil {
		logrus.Error(err)
		return err
	}

	bytes, err = ioutil.ReadAll(objectPtr)
	if err != nil {
		logrus.Error(err)
		return err
	}

	genericPrivateKey, err := crypto.ParseRSAPrivateKeyFromPEM(bytes)
	if err != nil {
		logrus.Error(err)
		return err
	}

	storeRootPrivateKey := asserts.RSAPrivateKey(rootPrivateKey)

	// create a signing database with the store's root key
	signingDB := assertstest.NewSigningDB("kebe-store", storeRootPrivateKey)

	// generate trusted account and account key
	createTrustedAccount(minioClient, storeRootPrivateKey, signingDB)

	// generate generic account, account-key and mode
	generateGenericAssertions(minioClient, storeRootPrivateKey, signingDB, asserts.RSAPrivateKey(genericPrivateKey), ctx, "generic")

	return nil
}

var Initialize = cobra.Command{
	Use:   "initialize",
	Short: "Initializes the store",

	Run: func(cmd *cobra.Command, args []string) {
		// Create root key
		minioClient := getMinioClient()
		storeRootPrivateKey := createRootKey(minioClient)

		// create a signing database with the store's root key
		signingDB := assertstest.NewSigningDB("kebe-store", storeRootPrivateKey)

		// generate trusted account and account key
		createTrustedAccount(minioClient, storeRootPrivateKey, signingDB)

		// generate generic account, account-key and mode
		createGenericAccount(minioClient, storeRootPrivateKey, signingDB)
	},
}

func getMinioClient() *minio.Client {
	accessKey := viper.GetString(configkey.MinioAccessKey)
	secretKey := viper.GetString(configkey.MinioSecretKey)
	minioHost := viper.GetString(configkey.MinioHost)

	logrus.Infof("Minio host=%s, accessKey=%s, secretKey=%s", minioHost, accessKey, secretKey)

	// Initialize minio client object.
	minioClient, err := minio.New(minioHost, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
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

func createGenericClassModelAssertion(signingDB *assertstest.SigningDB, keyId string, model string) *asserts.Model {
	modelHeaders := map[string]interface{}{
		"series":       "16",
		"brand-id":     "generic",
		"model":        "generic-classic",
		"timestamp":    "2015-11-20T15:04:00Z",
		"authority-id": "generic",
		"classic":      "true",
	}

	fmt.Printf("Sign generic model assertion with keyId: %s\n", keyId)
	a, err := signingDB.Sign(asserts.ModelType, modelHeaders, nil, keyId)

	if err != nil {
		logrus.Error(err)
	}

	return a.(*asserts.Model)
}

func createGenericAccount(minioClient *minio.Client, rootPrivateKey asserts.PrivateKey, signingDB *assertstest.SigningDB) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bucketName := "generic"
	err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		logrus.Error(err)
	}

	genericAccountKey, rsapk := crypto.CreateKeyPair(4096)
	keyString := crypto.ExportRsaPrivateKeyAsPemStr(rsapk)

	minioClient.PutObject(ctx, bucketName, "private-key.pem", strings.NewReader(keyString), int64(len(keyString)), minio.PutObjectOptions{})

	generateGenericAssertions(minioClient, rootPrivateKey, signingDB, genericAccountKey, ctx, bucketName)
}

func generateGenericAssertions(minioClient *minio.Client, rootPrivateKey asserts.PrivateKey, signingDB *assertstest.SigningDB, genericAccountKey asserts.PrivateKey, ctx context.Context, bucketName string) {
	signingDB.ImportKey(genericAccountKey)

	accountAssertion, bytes := createAccountAssertion(signingDB, rootPrivateKey.PublicKey().ID(), "generic", "generic")
	signingDB.Add(accountAssertion)

	_, err := minioClient.PutObject(ctx, bucketName, "account.assertion", strings.NewReader(string(bytes)), int64(len(bytes)), minio.PutObjectOptions{})
	if err != nil {
		logrus.Error(err)
	}

	_, bytes = createAccountKeyAssertion(signingDB, genericAccountKey.PublicKey(), rootPrivateKey.PublicKey().ID(), accountAssertion, "generic")
	_, err = minioClient.PutObject(ctx, bucketName, "account-key.assertion", strings.NewReader(string(bytes)), int64(len(bytes)), minio.PutObjectOptions{})
	if err != nil {
		logrus.Error(err)
	}

	modelAssertion := createGenericClassModelAssertion(signingDB, genericAccountKey.PublicKey().ID(), "generic")
	bytes = asserts.Encode(modelAssertion)
	if bytes == nil {
		logrus.Error("bytes is nil for model assertion!")
		return
	}
	_, err = minioClient.PutObject(ctx, bucketName, "model.assertion", strings.NewReader(string(bytes)), int64(len(bytes)), minio.PutObjectOptions{})
	if err != nil {
		logrus.Error(err)
	}
}

func createTrustedAccount(minioClient *minio.Client, rootPrivateKey asserts.PrivateKey, signingDB *assertstest.SigningDB) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	accountAssertion, bytes := createAccountAssertion(signingDB, rootPrivateKey.PublicKey().ID(), "kebe-store", "kebe-store")
	signingDB.Add(accountAssertion)

	_, err := minioClient.PutObject(ctx, "root", "account.assertion", strings.NewReader(string(bytes)), int64(len(bytes)), minio.PutObjectOptions{})
	if err != nil {
		logrus.Error(err)
	}

	_, bytes = createAccountKeyAssertion(signingDB, rootPrivateKey.PublicKey(), rootPrivateKey.PublicKey().ID(), accountAssertion, "kebe-store")
	_, err = minioClient.PutObject(ctx, "root", "account-key.assertion", strings.NewReader(string(bytes)), int64(len(bytes)), minio.PutObjectOptions{})
	if err != nil {
		logrus.Error(err)
	}
}

func createPrivateKey(minioClient *minio.Client, bucketName string, keyName string, bits int) asserts.PrivateKey {
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

	aPK, rsapk := crypto.CreateKeyPair(bits)
	keyString := crypto.ExportRsaPrivateKeyAsPemStr(rsapk)

	minioClient.PutObject(ctx, bucketName, keyName, strings.NewReader(keyString), int64(len(keyString)), minio.PutObjectOptions{})

	return aPK
}

func createRootKey(minioClient *minio.Client) asserts.PrivateKey {
	return createPrivateKey(minioClient, "root", "private-key.pem", 4096)
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
