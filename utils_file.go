package gocommonweb

import (
	"mime/multipart"
	"os"

	"github.com/h2non/filetype"
)

func getHeaderSection(fileHandle *multipart.FileHeader) ([]byte, error) {
	openedStream, err := fileHandle.Open()
	if err != nil {
		return nil, err
	}
	defer openedStream.Close()

	header := make([]byte, 261)
	openedStream.Read(header)
	return header, nil
}

// IsMultipartImageFile check if multiparth file header is an image file
func IsMultipartImageFile(fileHandle *multipart.FileHeader) (bool, error) {
	header, err := getHeaderSection(fileHandle)
	if err != nil {
		return false, err
	}
	return filetype.IsImage(header), nil
}

// MkDirIfNotExists create directory including children directory if not exists
func MkDirIfNotExists(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}

	return err
}

// IsFileExists check if given file path exists and not a directory
func IsFileExists(path string) bool {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !stat.IsDir()
}
