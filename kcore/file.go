package kcore

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
)

type File (string)

func NewFile(ext string) File {
	return File(NewID().String() + ext)
}

func (file File) Path() string {
	return fmt.Sprintf("media/files/%s", file)
}

func (file *File) Save(raw multipart.File) error {
	dest, err := os.Create(file.Path())
	if err != nil {
		return Wrap(err, "error creating file destination")
	}
	_, err = io.Copy(dest, raw)
	if err != nil {
		return Wrap(err, "error copying to file destination")
	}
	return nil
}

func (file *File) Delete() error {
	err := os.Remove(file.Path())
	if err != nil {
		return Wrap(err, "error deleting file")
	}
	return nil
}
