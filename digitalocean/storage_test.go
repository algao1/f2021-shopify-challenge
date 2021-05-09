package digitalocean

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	"github.com/algao1/imgrepo"
	"github.com/joho/godotenv"
)

func randomBytes(len int) []byte {
	token := make([]byte, len)
	rand.Read(token)
	return token
}

func loadImageStorageService() (*ImageStorage, error) {
	err := godotenv.Load("../.env")
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to load .env", err)
	}

	is, err := NewImageStorage(
		os.Getenv("SPACES_KEY"),
		os.Getenv("SPACES_SECRET"),
		os.Getenv("SPACES_ENDPOINT"),
		os.Getenv("SPACES_REGION"),
		os.Getenv("SPACES_BUCKET"),
	)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to create image storage service", err)
	}

	return is, nil
}

func TestUploadDownload(t *testing.T) {
	data := [][]byte{
		randomBytes(10),
		randomBytes(1000),
		randomBytes(100000),
	}

	tests := map[string]struct {
		image  *imgrepo.Image
		search string
		expect []byte
		err    error
	}{
		"small": {
			image:  &imgrepo.Image{Id: "__small", Raw: data[0]},
			search: "__small",
			expect: data[0],
		},
		"medium": {
			image:  &imgrepo.Image{Id: "__medium", Raw: data[1]},
			search: "__medium",
			expect: data[1],
		},
		"large": {
			image:  &imgrepo.Image{Id: "__large", Raw: data[2]},
			search: "__large",
			expect: data[2],
		},
	}

	is, err := loadImageStorageService()
	if err != nil {
		t.Fatal(err)
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if err = is.Upload(tc.image); err != nil {
				t.Fatal(err)
			}

			bt, err := is.Download(tc.search)
			if err != nil {
				t.Fatal(err)
			}

			if res := bytes.Compare(bt, tc.expect); res != 0 {
				t.Fatalf("unable to delete %s", tc.search)
			}
			is.Delete(tc.image.Id)
		})
	}
}
