package files

import (
	"io"
	d_errors "me/cobble/errors"
	"mime/multipart"
	"os"
	"slices"
	"strings"

	"github.com/mrz1836/go-sanitize"
)

func UploadFile(file *multipart.FileHeader, fileSize int64, fileTypes []string, folder string) (string, error) {
	// Open the file
	src, err := file.Open()
	if err != nil {
		return "", err
	}

	defer src.Close()

	if file.Size > fileSize {
		return "", d_errors.ErrFileTooLarge
	}

	var splitFileName = strings.Split(file.Filename, ".")

	if !slices.Contains(fileTypes, splitFileName[len(splitFileName)-1]) {
		return "", d_errors.ErrFileBadExtension
	}

	var safeFilename = sanitize.URL(sanitize.PathName(file.Filename))

	// Destination
	dst, err := os.Create("./files/" + folder + "/" + safeFilename)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	return "/files/" + folder + "/" + file.Filename, nil
}

func UploadVersionFile(file *multipart.FileHeader) (string, error) {
	return UploadFile(file, 5*1024*1024, []string{"zip"}, "versions")
}

func UploadResourcePackFile(file *multipart.FileHeader) (string, error) {
	return UploadFile(file, 50*1024*1024, []string{"zip"}, "resources")
}

func UploadIconFile(file *multipart.FileHeader) (string, error) {
	return UploadFile(file, 2*1024*1024, []string{"png", "jpg"}, "icons")
}
