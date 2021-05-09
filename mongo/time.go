package mongo

import (
	"fmt"
	"time"

	"github.com/algao1/imgrepo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetTime(img *imgrepo.Image) (time.Time, error) {
	_id, err := primitive.ObjectIDFromHex(img.Id)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q: %w", "unable to get objectId from Hex", err)
	}

	return _id.Timestamp(), nil
}
