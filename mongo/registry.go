package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/algao1/imgrepo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ImageRegistry struct {
	col     *mongo.Collection
	storage imgrepo.ImageStorage
}

var _ imgrepo.ImageRegistry = (*ImageRegistry)(nil)

// NewImageRegistry returns a ImageRegistry with the MongoDB collection configured.
func NewImageRegistry(store imgrepo.ImageStorage, uri, db, col string) (*ImageRegistry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := connect(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to create UserService", err)
	}

	return &ImageRegistry{
		col:     client.Database(db).Collection(col),
		storage: store,
	}, nil
}

func (ir *ImageRegistry) Upload(img *imgrepo.Image) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	img.Id = primitive.NewObjectID().Hex()

	_, err := ir.col.InsertOne(ctx, img)
	if err != nil {
		return fmt.Errorf("%q: %w", "unable to upload image to registry", err)
	}

	err = ir.storage.Upload(img)
	if err != nil {
		return fmt.Errorf("%q: %w", "unable to upload image to storage", err)
	}

	return nil
}

func (ir *ImageRegistry) Download(requester, id string) (*imgrepo.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var img imgrepo.Image
	err := ir.col.FindOne(ctx, bson.M{"_id": id}).Decode(&img)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to find file", err)
	}

	if img.Owner != requester && img.Access != imgrepo.Public {
		return nil, fmt.Errorf("unable to access file: %s", id)
	}

	raw, err := ir.storage.Download(id)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to download file", err)
	}

	img.Raw = raw

	return &img, nil
}

func (ir *ImageRegistry) List(size int, requester, lastId string) ([]*imgrepo.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Filters and pagination.
	filters := bson.M{
		"$or": bson.A{
			bson.M{"access": imgrepo.Public},
			bson.M{"owner": requester},
		},
	}
	if len(lastId) > 0 {
		filters["_id"] = bson.M{"$lt": lastId}
	}

	// Query options.
	var opts []*options.FindOptions
	opts = append(opts, options.Find().SetSort(bson.M{"_id": -1}))
	opts = append(opts, options.Find().SetLimit(int64(size)))

	// Fetch cursor.
	cursor, err := ir.col.Find(ctx, filters, opts...)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "cursor not found", err)
	}
	defer cursor.Close(ctx)

	var res []*imgrepo.Image
	for cursor.Next(context.Background()) {
		var img imgrepo.Image
		err = cursor.Decode(&img)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", "unable to complete query", err)
		}
		res = append(res, &img)
	}

	return res, nil
}
