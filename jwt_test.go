package gocommonweb

import (
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const privateKey = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAwt17WOB1BLMJU2zQRBfqr/suQrlWcnC29FxVuAFWmbT5Dzdf\niHdnm7IoayPQ36tS7L06JZ/D5sMMAqO0lh8mMHaWnNcY0/vqx45hT6SdfQlW1n0k\ntrPF0lq2NwktwUk/YU/uHzoFiUHBJv0IOUVoiyQPTbju80gWlHPBAW2nybxuqBUx\ndlWOkNo6GGxtbxe+2EnVWLcg/aOFjc+Yjc/dmcZTKEcQzn666mla7iYW7QtzmH7y\n/YC6TL2qJHId9c05u5gZMp0kQt8Hk7CHs+apTBvxZfMWHHeiihbWVhM3WCkWw9vw\nBqlS2g1e6eaVXf1OKNICPPbSGaQzq0KGpBdVrQIDAQABAoIBAQC2E6URYol0nqV0\nIhRny8EqNhT/m5W+0LrikPQ3PmjgrkyZfy/wn0FcJZfGpGyi0b5mHlmaljHCCTXL\nsZMnQmu4lNYeLo2wZY72b39Vn4bYMkXFnKEVZdzNtJzVx+nM1Ng34SuqWdiaa6pC\n9+MbQFNyz3rNMKN1BkKy64XIA+HniFnwndtHK+kOHboujj3SR/n8XuKW0yr7DTIZ\nY52fo1L4I+fjJTJZoWE3AdkZQt2ECYp0WObRiBY2Cl+pgGdeU9GXryrvtjRl++M2\nlF4c7L4R+gsuVHrLreK0y596eGSpOqJluBBOdDftdJqVjyNcGAUhmyCQoVKXJVQg\nz4VnD8gBAoGBAMdE9x3ayZz0Yqyii9Lx54B862kgctolyupntzaETyR02gME+Jp0\njrjXwKksazYtm1NWhyG61bhKXTQ4V/7OkmQ8RQby7HbphGvuci2LSeNqevGAZlH8\nAGA4YUFMZdcfI9fekI3KGf7DUHKn0Zgmdn576n5nsZmDbpjffEFozcuBAoGBAPpX\nh6TrTFTmKrI28zJstabvn8cDni80li5ChAtnDzxMcTzT5651jAfsjZIOIkJ1zCNG\nyWZtzJIDCxHskepzQJY0uT/nsXZuvYxVmPeCUKQbnvirzz+0AL0qeCTKKrqiAV4h\nmeLNjkggHe2ahLkTwqvqpD/O6HdgahlByNHWcpAtAoGAWyZmy1c7Bfqb8E/iGbnh\npVp+7HWVU8gZy6NpoRxgf1KcLullNnG+nzrBvCC/Yeb2t+ZKpdkqzcPmYm6rgbjI\nKeWPxZ/1Hmeu1RgbTk36nHYmirWrGDFnkpv3kMD7jK2H3cTG5rTdszVwZSHys+BU\nL4NLPkr8aDZArtj7o4fnKwECgYEA9u1NY3OgGAHrzVt8KJmn16B2RjQgXEmPdNOx\nMRoXog94hloyJRfW5p4CyjTcwBc3IviYgUr+RGtyCN0C1HGYHFCnBQzBM6Npnbl0\np3ZHmoeQF5JIW3puXCg+13L+EJbpqHKWOmss06GyQ4JtNazzEOXh2vp4u/9Cx+Tm\nc2wGFoUCgYBhMVFCHsgwd0XRTUxtgRdfHBYgS5ml9DvfhR5Yb7POTxa2Mxox+CFl\nIV9Tg23N2njbXsdGZ+44tgage1pZj+lUjHfHH7AbxppPuKHIe7ABpZwGCkzrRqEk\njPKzE+ilfTbeWnKG+4Umosa1KBzG8mUkWe1+YG4TxjcZJkQ58jVF/A==\n-----END RSA PRIVATE KEY-----\n"

func getDefConfig() JWTConfig {
	return JWTConfig{
		SignatureAlgo:      jwa.RS256,
		KeyEncryptAlgo:     jwa.RSA1_5,
		ContentEncryptAlgo: jwa.A128CBC_HS256,
		JWECompressAlgo:    jwa.NoCompress,
	}
}

func getClaims() map[string]interface{} {
	return map[string]interface{}{
		"id":           "01234",
		jwt.SubjectKey: "aris",
	}
}

func TestJwtUtilJWS(t *testing.T) {
	doTest(t, false)
}

func TestJwtUtilJWE(t *testing.T) {
	doTest(t, true)
}

func doTest(t *testing.T, enableJWE bool) {
	config := getDefConfig()
	config.EnableJWE = enableJWE

	jwtUtil, err := NewJWTUtilWithPEM(privateKey, config)
	require.NoError(t, err)
	require.NotNil(t, jwtUtil)

	now := time.Unix(time.Now().Unix(), 0).UTC()
	claims := getClaims()
	claims[jwt.IssuedAtKey] = now
	claims[jwt.ExpirationKey] = time.Now().Add(time.Second * 3)

	token, err := jwtUtil.GenerateJWT(claims)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	t.Logf("\n---- JWE Enabled: %t ----\n%s\n", config.EnableJWE, token)

	jwtToken, err := jwtUtil.VerifyJWT(token)
	require.NoError(t, err)

	iat, ok := jwtToken.Get(jwt.IssuedAtKey)
	require.True(t, ok)
	require.Equal(t, now, iat)

	id, ok := jwtToken.Get("id")
	require.True(t, ok)
	require.Equal(t, "01234", id)

	name, ok := jwtToken.Get(jwt.SubjectKey)
	require.True(t, ok)
	require.Equal(t, "aris", name)

	// wait until expire
	time.Sleep(time.Second * 3)
	jwtToken, err = jwtUtil.VerifyJWT(token)
	require.Error(t, err)
	require.Nil(t, jwtToken)

	t.Log(err)
}

func BenchmarkGenerateJWE(b *testing.B) {
	config := getDefConfig()
	config.EnableJWE = true

	jwtUtil, err := NewJWTUtilWithPEM(privateKey, config)
	if err != nil {
		b.Error(err)
	}

	claims := getClaims()
	for i := 0; i < b.N; i++ {
		_, _ = jwtUtil.GenerateJWT(claims)
	}
}

func BenchmarkGenerateJWS(b *testing.B) {
	jwtUtil, err := NewJWTUtilWithPEM(privateKey, getDefConfig())
	if err != nil {
		b.Error(err)
	}

	claims := getClaims()
	for i := 0; i < b.N; i++ {
		_, _ = jwtUtil.GenerateJWT(claims)
	}
}
