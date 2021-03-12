package gocommonweb

import (
	"io"
	"time"
)

type ObjectVisibility string

const (
	ObjectPrivate         ObjectVisibility = "private"
	ObjectPublicReadWrite ObjectVisibility = "public-read-write"
	ObjectPublicRead      ObjectVisibility = "public-read"
)

// Storage is an abstraction for persistence storage mechanism,
// remember that all object path used here should be specified
// relative to the root location configured for each implementation
type Storage interface {
	// Read return reader to stream data from source
	Read(objectPath string) (io.ReadCloser, error)

	// Put store source stream into
	Put(objectPath string, source io.Reader, visibility ObjectVisibility) error

	// Delete object by objectPath
	Delete(objectPaths ...string) error

	// URL return object url
	URL(objectPath string, isObjectPublic bool) (string, error)

	// Copy source to destination
	Copy(srcObjectPath string, dstObjectPath string) error

	// Size return object size
	Size(objectPath string) (int64, error)

	// LastModified return last modified time of object
	LastModified(objectPath string) (time.Time, error)

	// Exist check whether object exists
	Exist(objectPath string) (bool, error)

	SetVisibility(objectPath string, visibility ObjectVisibility) error

	GetVisibility(objectPath string) (ObjectVisibility, error)
}
