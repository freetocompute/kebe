package objectstore

import (
	"context"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"path"
)

type ObjectStore interface {
	SaveFileToBucket(bucket string, filePath string)
	GetFileFromBucket(bucket string, filePath string)
}

type Impl struct {
	MinioClient *minio.Client
}

func NewObjectStore() *Impl {
	return &Impl{MinioClient: GetMinioClient()}
}
func (obs *Impl) GetFileFromBucket(bucket string, filePath string) (*[]byte, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	base := path.Base(filePath)

	objectPtr, err := obs.MinioClient.GetObject(ctx, bucket, base, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(objectPtr)
	return &bytes, err
}

func (obs *Impl) SaveFileToBucket(bucket string, filePath string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exists, _ := obs.MinioClient.BucketExists(ctx, bucket)
	if !exists {
		err := obs.MinioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			logrus.Error(err)
		}
	}

	base := path.Base(filePath)

	uploadInfo, err := obs.MinioClient.FPutObject(ctx, bucket, base, filePath, minio.PutObjectOptions{})
	if err != nil {
		return err
	}

	logrus.Infof("%+v", uploadInfo)

	return nil
}

func GetMinioClient() *minio.Client {
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
