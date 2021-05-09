package imgrepo

// ADD image(s) to the repository:
// 		X: one / bulk / enormous amount of images
// 		X: private or public (permissions)
// 		X: secure uploading and stored images

// TODO:
// > Add a total #match to ListImages.
// > Probably should store Owner info alongside session token,
// 	 		address at a later date.
// > Refactor code involving server.go and client.go
//      consider moving them in proto/ to contain dependency

type Permission int

const (
	Public Permission = iota
	Private
)

// Image contains information about the image.
type Image struct {
	Id     string `bson:"_id" json:"_id,omitempty"`
	Name   string
	Owner  string
	Access Permission
	Raw    []byte `bson:"-"` // unused in registry
	Hash   uint64 // unimplemented
	Kind   int    // unimplemented
}

// Unimplemented feature.
type ImageComparator interface {
	SetHash(img *Image) error
	Difference(img *Image) (int, error)
}

// ImageStorage manages the storage of the raw image.
type ImageStorage interface {
	// Upload uploads the image to the blob storage.
	// Returns nil on success, and error otherwise.
	Upload(img *Image) error

	// Download downloads the image with the corresponding id.
	// Returns nil on success, and error otherwise.
	Download(id string) ([]byte, error)
}

// ImageRegistry manages access (upload/download/list) of images.
type ImageRegistry interface {
	// Upload generates an entry (with id) in the registry, and
	// uploads the image to the blob storage.
	// Returns nil on success, and error otherwise.
	Upload(img *Image) error

	// Download looks for the id in the registry, and downloads
	// the image if it exists.
	// Returns nil on success, and error otherwise.
	Download(requester, id string) (*Image, error)

	// List returns a list of images viewable by the requester.
	List(size int, requester string, lastId string) ([]*Image, error)
}

// UserService manages user account information, such as registering
// an account, and logging in.
type UserService interface {
	// Register registers an account.
	// Returns nil on success, and error otherwise.
	Register(username, password string) error

	// Login verifies that the account is valid.
	// Returns nil on success, and error otherwise.
	Login(username, password string) error
}

// SessionService manages user sessions.
type SessionService interface {
	// NewSession creates a session, and returns an UUID key.
	NewSession() (string, error)

	// IsSession checks if a session exists with the UUID key.
	IsSession(uuid string) error
}

type ImageClient interface {
	Register(username, password string) error
	Login(username, password string) error
	Upload(img *Image) error
	Download(id string) (*Image, error)
	List(lastId string) ([]*Image, error)
}
