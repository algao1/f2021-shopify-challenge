package mongo

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	"github.com/algao1/imgrepo"
	"github.com/google/go-cmp/cmp"
	"github.com/joho/godotenv"
)

type mockImageStorage struct {
	store map[string][]byte
}

func (m *mockImageStorage) Upload(img *imgrepo.Image) error {
	m.store[img.Id] = img.Raw
	return nil
}

func (m *mockImageStorage) Download(id string) ([]byte, error) {
	data, ok := m.store[id]
	if !ok {
		return nil, fmt.Errorf("unable to find file")
	}

	return data, nil
}

func randomBytes(len int) []byte {
	token := make([]byte, len)
	rand.Read(token)
	return token
}

func tmpImageRegistry(is imgrepo.ImageStorage) (*ImageRegistry, error) {
	err := godotenv.Load("../.env")
	if err != nil {
		panic(err)
	}

	return NewImageRegistry(is, os.Getenv("MONGO_URI"), os.Getenv("MONGO_DB"), "_test")
}

func TestImageUploadDownload(t *testing.T) {
	tests := map[string]struct {
		requester string
		want      *imgrepo.Image
		expectErr bool
	}{
		"owner access public": {
			requester: "test",
			want:      &imgrepo.Image{Owner: "test", Access: imgrepo.Public, Raw: randomBytes(1000)},
			expectErr: false,
		},
		"owner access private": {
			requester: "test",
			want:      &imgrepo.Image{Owner: "test", Access: imgrepo.Private, Raw: randomBytes(1000)},
			expectErr: false,
		},
		"other access public": {
			requester: "test2",
			want:      &imgrepo.Image{Owner: "test", Access: imgrepo.Public, Raw: randomBytes(1000)},
			expectErr: false,
		},
		"other access private": {
			requester: "test2",
			want:      &imgrepo.Image{Owner: "test", Access: imgrepo.Private, Raw: randomBytes(1000)},
			expectErr: true,
		},
	}

	ir, err := tmpImageRegistry(&mockImageStorage{store: make(map[string][]byte)})
	if err != nil {
		t.Fatal(err)
	}
	defer ir.col.Drop(context.TODO())

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err = ir.Upload(tc.want)
			if err != nil && !tc.expectErr {
				t.Fatal(err)
			}

			got, err := ir.Download(tc.requester, tc.want.Id)
			if err != nil && !tc.expectErr {
				t.Fatal(err)
			} else if diff := cmp.Diff(tc.want, got); !tc.expectErr && diff != "" {
				t.Fatalf("Upload() and Download() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func cmpSlices(s1, s2 []*imgrepo.Image) error {
	if len(s1) != len(s2) {
		return fmt.Errorf("slices have varying lengths")
	}

	for idx := range s1 {
		if !cmp.Equal(s1[idx], s2[idx]) {
			return fmt.Errorf("slices differ at %d: %v and %v", idx, s1[idx], s2[idx])
		}
	}

	return nil
}

func TestListImages(t *testing.T) {
	images := []*imgrepo.Image{
		{Owner: "test", Access: imgrepo.Public},
		{Owner: "test", Access: imgrepo.Private},
		{Owner: "test2", Access: imgrepo.Public},
		{Owner: "test2", Access: imgrepo.Private},
		{Owner: "test3", Access: imgrepo.Private},
		{Owner: "test3", Access: imgrepo.Private},
		{Owner: "test4", Access: imgrepo.Public},
		{Owner: "test4", Access: imgrepo.Public},
	}

	tests := map[string]struct {
		requester string
		want      []int
	}{
		"owner: test": {
			requester: "test",
			want:      []int{0, 1, 2, 6, 7},
		},
		"owner: test2": {
			requester: "test2",
			want:      []int{0, 2, 3, 6, 7},
		},
		"owner: test3": {
			requester: "test3",
			want:      []int{0, 2, 4, 5, 6, 7},
		},
	}

	ir, err := tmpImageRegistry(&mockImageStorage{store: make(map[string][]byte)})
	if err != nil {
		t.Fatal(err)
	}
	defer ir.col.Drop(context.TODO())

	for idx := range images {
		err := ir.Upload(images[len(images)-idx-1])
		if err != nil {
			t.Fatal(err)
		}
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ir.List(20, tc.requester, "")
			if err != nil {
				t.Fatal(err)
			}

			want := make([]*imgrepo.Image, len(tc.want))
			for i, idx := range tc.want {
				want[i] = images[idx]
			}

			if err := cmpSlices(want, got); err != nil {
				t.Fatal(err)
			}
		})
	}
}
