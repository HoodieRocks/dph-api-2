package files

import (
	"archive/zip"
	"io"
	"mime/multipart"
	"os"
	"slices"
	"strings"

	derrors "github.com/HoodieRocks/dph-api-2/errors"
	"github.com/HoodieRocks/dph-api-2/utils/db"

	"github.com/h2non/bimg"
	"github.com/mrz1836/go-sanitize"
)

func UploadFile(file *multipart.FileHeader, maxSize int64, fileTypes []string, filename string, folder string) (*os.File, error) {
	// Open the file
	src, err := file.Open()
	if err != nil {
		return nil, err
	}

	defer src.Close()

	if file.Size > maxSize {
		return nil, derrors.ErrFileTooLarge
	}

	var splitFileName = strings.Split(file.Filename, ".")
	var fileExt = strings.ToLower(splitFileName[len(splitFileName)-1])

	if !slices.Contains(fileTypes, fileExt) {
		return nil, derrors.ErrFileBadExtension
	}

	var safeFilename = sanitize.PathName(file.Filename)

	// Destination
	dst, err := os.Create("./files/" + folder + "/" + strings.TrimSuffix(safeFilename, "zip") + "." + fileExt)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return nil, err
	}

	return dst, nil
}

func UploadZipFile(file *multipart.FileHeader, maxSize int64, folder string) (string, error) {

	var safeFilename = sanitize.PathName(file.Filename)
	dst, err := UploadFile(file, maxSize, []string{"zip"}, file.Filename, folder)

	if err != nil {
		return "", err
	}

	newHandle, _ := os.Open(dst.Name())
	fs, err := newHandle.Stat()
	defer newHandle.Close()

	if err != nil {
		return "", err
	}

	header, err := zip.FileInfoHeader(fs)

	if err != nil {
		return "", err
	}

	if header.UncompressedSize64 > uint64(maxSize*2) {
		err = os.Remove("./files/" + folder + "/" + safeFilename)

		if err != nil {
			return "", err
		}

		return "", derrors.ErrFileTooLarge
	}

	return "/files/" + folder + "/" + file.Filename, nil
}

func UploadVersionFile(file *multipart.FileHeader, project db.Project) (string, error) {
	return UploadZipFile(file, 5*1024*1024, "versions/"+project.Slug)
}

func UploadResourcePackFile(file *multipart.FileHeader, project db.Project) (string, error) {
	return UploadZipFile(file, 50*1024*1024, "resources/"+project.Slug)
}

func UploadIconFile(file *multipart.FileHeader, project db.Project) (string, error) {
	_, err := UploadFile(file, 2*1024*1024, []string{"png", "jpg"}, project.Slug+"png", "icons")

	if err != nil {
		return "", err
	}

	//TODO security hotspot path traversal attack possible?
	//TODO dot missing?
	buffer, err := bimg.Read("./files/icons/" + project.Slug + "png")

	if err != nil {
		return "", err
	}

	smallImg, _ := bimg.NewImage(buffer).Resize(256, 256)
	img, err := bimg.NewImage(smallImg).Convert(bimg.WEBP)

	if err != nil {
		return "", err
	}

	err = bimg.Write("./files/icons/"+project.Slug+".webp", img)

	if err != nil {
		return "", err
	}

	go os.Remove("./files/icons/" + project.Slug + "png")

	return "/files/icons/" + project.Slug + ".webp", nil
}
