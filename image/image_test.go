package image

// import (
// 	"os"
// 	"strings"
// 	"testing"

// 	"github.com/algao1/imgrepo"
// 	"github.com/google/go-cmp/cmp"
// )

// func TestNewImageInvalidFiles(t *testing.T) {
// 	tests := map[string]struct {
// 		name   string
// 		owner  string
// 		access imgrepo.Permission
// 		reason string
// 	}{
// 		"fileNotFound": {
// 			name:   "fileNotFound.png",
// 			owner:  "test",
// 			access: imgrepo.Public,
// 			reason: "unable to open file",
// 		},
// 		"invalidFileFormat": {
// 			name:   "invalidFileFormat.bmp",
// 			owner:  "test",
// 			access: imgrepo.Public,
// 			reason: "invalid image format: .bmp",
// 		},
// 		"invalidDecode": {
// 			name:   "emptyImage.jpg",
// 			owner:  "test",
// 			access: imgrepo.Public,
// 			reason: "unable to decode file",
// 		},
// 	}

// 	os.Create("invalidFileFormat.bmp")
// 	os.Create("emptyImage.jpg")
// 	defer func() {
// 		os.Remove("invalidFileFormat.bmp")
// 		os.Remove("emptyImage.jpg")
// 	}()

// 	for name, tc := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			_, err := NewImage(tc.name, tc.owner, tc.access)
// 			if err != nil && !strings.Contains(err.Error(), tc.reason) {
// 				t.Fatalf(cmp.Diff(err.Error(), tc.reason))
// 			}
// 		})
// 	}
// }
