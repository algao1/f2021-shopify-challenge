package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/algao1/imgrepo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	col *mongo.Collection
}

var _ imgrepo.UserService = (*UserService)(nil)

type Credentials struct {
	Username string
	Password []byte
}

func connect(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to connect to collection", err)
	}

	return client, nil
}

// NewUserService returns a UserService with the MongoDB collection configured.
func NewUserService(uri, db, col string) (*UserService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := connect(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to create UserService", err)
	}

	return &UserService{col: client.Database(db).Collection(col)}, nil
}

func (us *UserService) Register(user, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cred Credentials

	err := us.col.FindOne(ctx, bson.M{"username": user}).Decode(&cred)
	if err == nil {
		return fmt.Errorf("username already exists: %s", user)
	} else if err != mongo.ErrNoDocuments {
		return fmt.Errorf("%q: %w", "unexpected error", err)
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return fmt.Errorf("%q: %w", "unable to encrypt password", err)
	}

	_, err = us.col.InsertOne(ctx, bson.M{"username": user, "password": bytes})
	if err != nil {
		return fmt.Errorf("%q: %w", "unable to register user", err)
	}

	return nil
}

func (us *UserService) Login(user, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cred Credentials

	err := us.col.FindOne(ctx, bson.M{"username": user}).Decode(&cred)
	if err == mongo.ErrNoDocuments {
		return fmt.Errorf("incorrect username or password")
	} else if err != nil {
		return fmt.Errorf("%q: %w", "unexpected error", err)
	}

	err = bcrypt.CompareHashAndPassword(cred.Password, []byte(password))
	if err != nil {
		return fmt.Errorf("incorrect username or password")
	}

	return nil
}
