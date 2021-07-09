package objectstore

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/sha"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
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

	sha3_384, size, _ := sha.SnapFileSHA3_384FromReader(objectPtr)
	logrus.Infof("bucket: %s, object name: %s, sha3_384: %s, size: %d", bucket, filePath, sha3_384, size)

	objectPtr, err = obs.MinioClient.GetObject(ctx, bucket, base, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	h := crypto.SHA3_384.New()
	objectPtr, err = obs.MinioClient.GetObject(ctx, bucket, base, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	bytes, _ := io.ReadAll(objectPtr)
	h.Sum(bytes)
	actualSha3 := fmt.Sprintf("%x", h.Sum(nil))
	logrus.Infof("Actual sha3: %s", actualSha3)

	//bytes, err := ioutil.ReadAll(objectPtr)
	return &bytes, err
}

func (obs *Impl) Move(sourceBucket, destinationBucket, objectName string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sourceBucketExists, err := obs.MinioClient.BucketExists(ctx, sourceBucket)
	if err != nil {
		return err
	}
	destinationBucketExists, err := obs.MinioClient.BucketExists(ctx, destinationBucket)
	if err != nil {
		return err
	}

	if sourceBucketExists && destinationBucketExists {
		_, err := obs.MinioClient.CopyObject(ctx, minio.CopyDestOptions{Bucket: destinationBucket, Object: objectName}, minio.CopySrcOptions{Bucket: sourceBucket, Object: objectName})
		if err != nil {
			return err
		}

		return nil
	}

	return errors.New("something went wrong")
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

	h := crypto.SHA3_384.New()
	objectPtr, err := obs.MinioClient.GetObject(ctx, bucket, base, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	bytes, _ := io.ReadAll(objectPtr)
	h.Sum(bytes)
	actualSha3 := fmt.Sprintf("%x", h.Sum(nil))
	logrus.Infof("Actual sha3: %s", actualSha3)

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
