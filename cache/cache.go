package cache

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Cache struct {
	S3BucketName string
}

func (c *Cache) Get(key string) ([]byte, error) {
	if c.S3BucketName != "" {
		return c.S3Get(key)
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

	svc := s3.New(session.New())

	params := &s3.GetObjectInput{
		Bucket: aws.String(c.S3BucketName),
		Key:    aws.String(key),
	}

	resp, err := svc.GetObject(params)

	if err != nil {
		return image, err
	}

	image, err = ioutil.ReadAll(resp.Body)

	return image, nil
}

func (c *Cache) Set(key string, value []byte) error {
	if c.S3BucketName != "" {
		return c.S3Set(key, value)
	}

	return c.FSSet(key, value)
}

func (c *Cache) S3Set(key string, value []byte) error {

	if len(value) == 0 {
		return nil
	}

	if _, err := c.S3Get(key); err == nil {
		return nil
	}

	svc := s3.New(session.New())
	params := &s3.PutObjectInput{
		Bucket: aws.String(c.S3BucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(value),
	}

	_, err := svc.PutObject(params)

	return err
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

func (c *Cache) Delete(key string) error {
	return nil
}
