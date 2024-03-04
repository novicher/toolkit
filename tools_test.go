package toolkit

import (
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)

	if len(s) != 10 {
		t.Errorf("Expected random string of length 10, but got length: %d", len(s))
	}
}

var uploadTests = []struct {
	name          string
	allowedtypes  []string
	renameFile    bool
	errorExpected bool
}{
	{"allowed no-rename", []string{"image/jpeg", "image/png"}, false, false},
	{"allowed with rename", []string{"image/jpeg", "image/png"}, true, false},
	{"not allowed", []string{"application/pdf", "image/jpeg"}, true, true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// setup pipe
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer writer.Close()
			defer wg.Done()

			// create form data
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error("failed to create part from file img.png", err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error("failed to open file img.png", err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error("failed to encode png", err)
			}
		}()

		// read from file which receives data
		req := httptest.NewRequest("POST", "/", pr)
		req.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools = Tools{
			AllowedFileTypes: e.allowedtypes,
		}

		files, err := testTools.UploadFiles(req, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if err == nil && e.errorExpected {
			t.Errorf("%s: expected error but none received", e.name)
		}

		if !e.errorExpected {
			f := files[0]
			fileName := filepath.Join("./testdata/uploads", f.NewFileName)
			_, err = os.Stat(fileName)
			if err != nil {
				t.Errorf("%s: expected to find file %s", e.name, err.Error())
			}
			os.Remove(fileName)

			if e.renameFile {
				if f.OriginalFileName == f.NewFileName {
					t.Error("expected file to be renamed")
				}
			} else {
				if f.OriginalFileName != f.NewFileName {
					t.Error("expected file to retain name")
				}
			}

		}

	}
}
