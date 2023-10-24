package kcore

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
)

type Image (ID)

func NewImage() Image {
	return Image(NewID())
}

func (image Image) Path() string {
	return fmt.Sprintf("media/images/%s", image)
}

func (image *Image) Save(raw multipart.File) error {
	dest, err := os.Create(image.Path())
	if err != nil {
		return Wrap(err, "error creating image destination")
	}
	_, err = io.Copy(dest, raw)
	if err != nil {
		return Wrap(err, "error copying to image destination")
	}
	return nil
}

func (image *Image) Delete() error {
	err := os.Remove(image.Path())
	if err != nil {
		return Wrap(err, "error deleting image")
	}
	return nil
}
