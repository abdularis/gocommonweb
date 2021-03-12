package gocommonweb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPasswordHashing(t *testing.T) {
	password := "my_password_1234"

	hashed, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashed)

	valid := CheckPasswordHash(password, hashed)
	require.Equal(t, true, valid)

	valid = CheckPasswordHash("my_password", hashed)
	require.Equal(t, false, valid)
}
