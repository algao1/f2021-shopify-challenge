package mongo

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func tmpUserService() (*UserService, error) {
	err := godotenv.Load("../.env")
	if err != nil {
		panic(err)
	}

	return NewUserService(os.Getenv("MONGO_URI"), os.Getenv("MONGO_DB"), "_test")
}

func TestRegisterUser(t *testing.T) {
	tests := map[string]struct {
		username  string
		password  string
		expectErr bool
	}{
		"register new user":       {username: "nadmin", password: "password", expectErr: false},
		"register duplicate user": {username: "admin", password: "password", expectErr: true},
	}

	us, err := tmpUserService()
	if err != nil {
		t.Fatal(err)
	}
	defer us.col.Drop(context.TODO())

	// Setup existing user account.
	us.Register("admin", "password")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err = us.Register(tc.username, tc.password)
			if err != nil && !tc.expectErr {
				t.Fatal(err)
			} else if err == nil && tc.expectErr {
				t.Fatal(err)
			}
		})
	}
}

func TestLoginUser(t *testing.T) {
	tests := map[string]struct {
		username  string
		password  string
		expectErr bool
	}{
		"invalid username":        {username: "admin2", password: "password", expectErr: true},
		"invalid password":        {username: "admin", password: "password2", expectErr: true},
		"valid username/password": {username: "admin", password: "password", expectErr: false},
	}

	us, err := tmpUserService()
	if err != nil {
		t.Fatal(err)
	}
	defer us.col.Drop(context.TODO())

	// Setup existing user account.
	us.Register("admin", "password")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err = us.Login(tc.username, tc.password)
			if err != nil && !tc.expectErr {
				t.Fatal(err)
			} else if err == nil && tc.expectErr {
				t.Fatal("expected error")
			}
		})
	}
}
