package cache

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/Azure/azure-sdk-for-go/storage"
)

type Cache struct {
	S3BucketName       string
	AzureContainerName string
	S3Client           *s3.S3
	AzureClient        storage.BlobStorageClient
}

func NewCache(s3Bucket, azureContainer string) (*Cache, error) {
	c := Cache{
		S3BucketName:       s3Bucket,
		AzureContainerName: azureContainer,
	}

	if s3Bucket != "" {
		c.S3Client = s3.New(session.New())
	} else if azureContainer != "" {
		accountName := os.Getenv("AZURE_ACCOUNT_NAME")
		accountKey := os.Getenv("AZURE_ACCOUNT_KEY")
		azureBasicCli, err := storage.NewBasicClient(accountName, accountKey)
		if err != nil {
			return nil, err
		}

		c.AzureClient = azureBasicCli.GetBlobService()
	}

	return &c, nil
}

func (c *Cache) Get(key string) ([]byte, error) {
	if c.S3BucketName != "" {
		return c.S3Get(key)
	}

	if c.AzureContainerName != "" {
		return c.AzureGet(key)
	}

	return c.FSGet(key)
}

func (c *Cache) FSGet(key string) ([]byte, error) {

	var image []byte
	var err error

	if _, err = os.Stat(key); os.IsNotExist(err) {
		return image, err
	}

	file, err := os.Open(key)
	defer file.Close()

	if err != nil {
		return image, err
	}

	info, _ := file.Stat()
	image = make([]byte, info.Size())

	_, err = file.Read(image)
	if err != nil {
		return image, err
	}

	return image, nil
}

func (c *Cache) S3Get(key string) ([]byte, error) {

	var image []byte
	var err error

	params := &s3.GetObjectInput{
		Bucket: aws.String(c.S3BucketName),
		Key:    aws.String(key),
	}

	resp, err := c.S3Client.GetObject(params)

	if err != nil {
		return image, err
	}
	defer resp.Body.Close()

	image, err = ioutil.ReadAll(resp.Body)

	return image, nil
}

func (c *Cache) AzureGet(key string) ([]byte, error) {

	var image []byte
	var err error

	rc, err := c.AzureClient.GetBlob(c.AzureContainerName, key)

	if err != nil {
		return image, err
	}
	defer rc.Close()

	image, err = ioutil.ReadAll(rc)

	return image, nil
}

func (c *Cache) Set(key string, value []byte) error {
	if c.S3BucketName != "" {
		return c.S3Set(key, value)
	}

	if c.AzureContainerName != "" {
		return c.AzureSet(key, value)
	}

	return c.FSSet(key, value)
}

func (c *Cache) FSSet(key string, value []byte) error {

	if len(value) == 0 {
		return nil
	}

	if _, err := os.Stat(key); err == nil {
		return nil
	}

	err := os.MkdirAll(path.Dir(key), 0777)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(key, value, 0666)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cache) S3Set(key string, value []byte) error {

	if len(value) == 0 {
		return nil
	}

	if _, err := c.S3Get(key); err == nil {
		return nil
	}

	params := &s3.PutObjectInput{
		Bucket: aws.String(c.S3BucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(value),
	}

	_, err := c.S3Client.PutObject(params)

	return err
}

func (c *Cache) AzureSet(key string, value []byte) error {

	if len(value) == 0 {
		return nil
	}

	if exists, _ := c.AzureClient.BlobExists(c.AzureContainerName, key); exists == true {
		return nil
	}

	reader := bytes.NewReader(value)

	err := c.AzureClient.CreateBlockBlobFromReader(c.AzureContainerName,
		key, uint64(len(value)), reader, map[string]string{})

	return err
}

func (c *Cache) Delete(key string) error {
	return nil
}
