package image

// import (
// 	"fmt"
// 	"image"
// 	"image/jpeg"
// 	"image/png"
// 	"os"
// 	"regexp"

// 	"github.com/algao1/imgrepo"
// 	"github.com/corona10/goimagehash"
// )

// type Image struct {
// 	name    string
// 	owner   string
// 	access  imgrepo.Permission
// 	data    *image.Image
// 	rawHash uint64
// 	kind    int
// }

// var _ imgrepo.Image = (*Image)(nil)

// // Temporary global regex, remove later.
// var _fmtReg *regexp.Regexp = regexp.MustCompile(`\..+$`)

// func NewImage(name, owner string, access imgrepo.Permission) (*Image, error) {
// 	file, err := os.Open(name)
// 	if err != nil {
// 		return nil, fmt.Errorf("%q: %w", "unable to open file", err)
// 	}
// 	defer file.Close()

// 	format := _fmtReg.FindString(name)

// 	var img image.Image

// 	if format == ".jpg" {
// 		img, err = jpeg.Decode(file)
// 	} else if format == ".png" {
// 		img, err = png.Decode(file)
// 	} else {
// 		return nil, fmt.Errorf("invalid image format: %s", format)
// 	}

// 	if err != nil {
// 		return nil, fmt.Errorf("%q: %w", "unable to decode file", err)
// 	}

// 	hash, err := goimagehash.AverageHash(img)
// 	if err != nil {
// 		return nil, fmt.Errorf("%q: %w", "unable to hash image", err)
// 	}

// 	return &Image{
// 		name:    name,
// 		owner:   owner,
// 		access:  access,
// 		data:    &img,
// 		rawHash: hash.GetHash(),
// 		kind:    int(hash.GetKind()),
// 	}, nil
// }

// func (img *Image) Name() string {
// 	return img.name
// }

// func (img *Image) Hash() (uint64, int) {
// 	return img.rawHash, img.kind
// }

// func (img *Image) Difference(target *imgrepo.Image) (int, error) {
// 	rhash1, kind1 := img.Hash()
// 	rhash2, kind2 := target.Hash()

// 	hash1 := goimagehash.NewImageHash(rhash1, goimagehash.Kind(kind1))
// 	hash2 := goimagehash.NewImageHash(rhash2, goimagehash.Kind(kind2))

// 	return hash1.Distance(hash2)
// }
