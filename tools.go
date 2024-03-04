package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const randomStringSource = "abcdefghigklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is used to instantiate this module.
type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string
}

// RandomString returns a string of random characters of length 'n'
func (t *Tools) RandomString(n int) string {

	s, r := make([]rune, n), []rune(randomStringSource)

	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}

	return string(s)
}

// UploadedFile is used to return metadata about the file
type UploadedFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int
}

func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {

	file, err := t.UploadFiles(r, uploadDir, rename...)
	if err != nil {
		return nil, err
	}
	return file[0], nil
}
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {

	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024 // 1GB
	}

	err := r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("the uploaded file is too big")
	}

	err = t.CreateDirIfNotExists(uploadDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create uploadDir: %s", uploadDir)
	}

	for _, fh := range r.MultipartForm.File {
		for _, hdr := range fh {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {

				var uploadedFile UploadedFile

				infile, err := hdr.Open()
				if err != nil {
					return uploadedFiles, err
				}
				defer infile.Close()

				// check size
				if hdr.Size > int64(t.MaxFileSize) {
					return nil, fmt.Errorf("file %s of size %d exceeds max size of %d", hdr.Filename, hdr.Size, t.MaxFileSize)
				}

				// check mime type
				buff := make([]byte, 512)
				_, err = infile.Read(buff)
				if err != nil {
					return uploadedFiles, err
				}
				var allowed = false
				fileType := http.DetectContentType(buff)
				if len(t.AllowedFileTypes) > 0 {
					for _, pt := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, pt) {
							allowed = true
							break
						}
					}
				} else {
					allowed = true
				}
				if !allowed {
					return uploadedFiles, fmt.Errorf("uploaded file type: %s is not permitted", fileType)
				}
				_, err = infile.Seek(0, 0)
				if err != nil {
					return uploadedFiles, err
				}

				// check / generate new name
				uploadedFile.OriginalFileName = hdr.Filename
				if renameFile {
					// generate new name
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(20), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFileName = uploadedFile.OriginalFileName
				}

				// save file
				var outfile *os.File
				defer outfile.Close()

				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return uploadedFiles, err
				}
				size, err := io.Copy(outfile, infile)
				if err != nil {
					return uploadedFiles, err
				}
				uploadedFile.FileSize = int(size)

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil
			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}
	}
	return uploadedFiles, err
}

// CreateDirIfNotExists creates a directory if it does not already exist
func (t *Tools) CreateDirIfNotExists(path string) error {
	const mode = 0755
	if _, err := os.Stat(path); err != nil {
		err = os.MkdirAll(path, mode)
		return err
	}
	return nil
}
