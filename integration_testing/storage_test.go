package integration_testing

import (
	"gocommonweb"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestStorageS3Suite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip test for storage aws s3")
	}

	testSuite := &storageTestSuite{
		storage: gocommonweb.
			NewStorageAWSS3(
				"go-integration-test",
				"ap-southeast-1",
				"access-key-id",
				"secret-access-key",
				""),
	}

	suite.Run(t, testSuite)
}

func TestStorageLocalSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip test for local file storage")
	}

	testSuite := &storageTestSuite{
		storage: gocommonweb.NewStorageLocalFile("storage/private", "storage/public", "http://localhost:8080/files"),
	}

	suite.Run(t, testSuite)
}

type storageTestSuite struct {
	suite.Suite
	storage gocommonweb.Storage
}

func (s *storageTestSuite) TestPutAndGet() {
	localContent := "hello there this is sample small file for storage testing"

	err := s.storage.Put("test/readme.txt", strings.NewReader(localContent), gocommonweb.ObjectPrivate)
	require.NoError(s.T(), err)

	reader, err := s.storage.Read("test/readme.txt")
	require.NoError(s.T(), err)

	content, err := ioutil.ReadAll(reader)
	require.NoError(s.T(), err)

	require.Equal(s.T(), localContent, string(content))
}

func (s *storageTestSuite) TestPutAndDelete() {
	c1 := "Hello this is first content"
	c2 := "Hello this is second content"

	err := s.storage.Put("test/file1.txt", strings.NewReader(c1), gocommonweb.ObjectPrivate)
	require.NoError(s.T(), err)
	err = s.storage.Put("test/file2.txt", strings.NewReader(c2), gocommonweb.ObjectPrivate)
	require.NoError(s.T(), err)

	err = s.storage.Delete("test/file1.txt", "test/file2.txt")
	require.NoError(s.T(), err)
}
