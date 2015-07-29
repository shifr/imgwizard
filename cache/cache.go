package cache

import (
	"io/ioutil"
	"os"
	"path"
)

type Cache struct{}

func (c *Cache) Get(key string) ([]byte, error) {
	var image []byte

	file, err := os.Open(key)
	if err != nil {
		return image, err
	}
	defer file.Close()

	info, _ := file.Stat()
	image = make([]byte, info.Size())

	_, err = file.Read(image)
	if err != nil {
		return image, err
	}

	return image, nil
}

func (c *Cache) Set(key string, value []byte) error {

	if len(value) == 0 {
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
