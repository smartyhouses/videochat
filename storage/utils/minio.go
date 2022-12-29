package utils

import (
	"context"
	"errors"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	. "nkonev.name/storage/logger"
	"strings"
)

func ensureBucket(minioClient *minio.Client, bucketName, location string) error {
	// Check to see if we already own this bucket (which happens if you run this twice)
	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err == nil && exists {
		Logger.Infof("Bucket '%s' already present", bucketName)
		return nil
	} else if err != nil {
		return err
	} else {
		if err := minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{
			Region:        location,
			ObjectLocking: false,
		}); err != nil {
			Logger.Errorf("Error during creating bucket '%s'", bucketName)
			return err
		} else {
			Logger.Infof("Successfully created bucket '%s'", bucketName)
			return nil
		}
	}
}

func EnsureAndGetUserAvatarBucket(minioClient *minio.Client) (string, error) {
	bucketName := viper.GetString("minio.bucket.userAvatar")
	bucketLocation := viper.GetString("minio.location")
	err := ensureBucket(minioClient, bucketName, bucketLocation)
	return bucketName, err
}

func EnsureAndGetChatAvatarBucket(minioClient *minio.Client) (string, error) {
	bucketName := viper.GetString("minio.bucket.chatAvatar")
	bucketLocation := viper.GetString("minio.location")
	err := ensureBucket(minioClient, bucketName, bucketLocation)
	return bucketName, err
}

func EnsureAndGetFilesBucket(minioClient *minio.Client) (string, error) {
	bucketName := viper.GetString("minio.bucket.files")
	bucketLocation := viper.GetString("minio.location")
	err := ensureBucket(minioClient, bucketName, bucketLocation)
	return bucketName, err
}

func EnsureAndGetEmbeddedBucket(minioClient *minio.Client) (string, error) {
	bucketName := viper.GetString("minio.bucket.embedded")
	bucketLocation := viper.GetString("minio.location")
	err := ensureBucket(minioClient, bucketName, bucketLocation)
	return bucketName, err
}

func EnsureAndGetFilesPreviewBucket(minioClient *minio.Client) (string, error) {
	bucketName := viper.GetString("minio.bucket.filesPreview")
	bucketLocation := viper.GetString("minio.location")
	err := ensureBucket(minioClient, bucketName, bucketLocation)
	return bucketName, err
}

type MinioConfig struct {
	UserAvatar, ChatAvatar, Files, Embedded, FilesPreview string
}

const ObjectCreated = "s3:ObjectCreated"
const ObjectRemoved = "s3:ObjectRemoved"

func FilesIdToFilesPreviewId(key string, minioConfig *MinioConfig) string {
	// transforms "files/chat/116/ad36c70a-c9ae-4846-9c25-6d5f5ac94873/561ae246-7eff-45a6-a480-2b2be254c768.jpg" to
	// "files-preview/chat/116/ad36c70a-c9ae-4846-9c25-6d5f5ac94873/561ae246-7eff-45a6-a480-2b2be254c768.jpg"
	return strings.ReplaceAll(key, minioConfig.Files, minioConfig.FilesPreview)
}

func SetVideoPreviewExtension(key string) string {
	return SetExtension(key, "png")
}

func SetImagePreviewExtension(key string) string {
	return SetExtension(key, "jpg")
}

const FileParam = "file"

func ParseChatId(minioKey string) (int64, error) {
	// "chat/116/ad36c70a-c9ae-4846-9c25-6d5f5ac94873/561ae246-7eff-45a6-a480-2b2be254c768.jpg"
	split := strings.Split(minioKey, "/")
	if len(split) >= 2 {
		str := split[1]
		return ParseInt64(str)
	}
	return 0, errors.New("Unable to parse file id")
}

func StripBucketName(minioKey string, bucketName string) string {
	// "files/chat/116/ad36c70a-c9ae-4846-9c25-6d5f5ac94873/561ae246-7eff-45a6-a480-2b2be254c768.jpg"
	toStrip := bucketName + "/"
	return strings.ReplaceAll(minioKey, toStrip, "")
}
