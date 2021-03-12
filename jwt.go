package gocommonweb

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwe"
	"github.com/lestrrat-go/jwx/jwt"
)

type JWTConfig struct {
	SignatureAlgo      jwa.SignatureAlgorithm
	EnableJWE          bool
	KeyEncryptAlgo     jwa.KeyEncryptionAlgorithm
	ContentEncryptAlgo jwa.ContentEncryptionAlgorithm
	JWECompressAlgo    jwa.CompressionAlgorithm
}

// JWTUtil generate and verify JWT
type JWTUtil struct {
	privateKey *rsa.PrivateKey
	config     JWTConfig
}

// NewJWTUtil create new Jwt verification
func NewJWTUtil(privateKeyPath string, config JWTConfig) (*JWTUtil, error) {
	file, err := os.Open(privateKeyPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if privateKey, err := getRSAPrivateKey(data); err != nil {
		return nil, err
	} else {
		return &JWTUtil{
			privateKey: privateKey,
			config:     config,
		}, nil
	}
}

// NewJWTUtilWithPEM create new jwt verification with provided private key
func NewJWTUtilWithPEM(pem string, config JWTConfig) (*JWTUtil, error) {
	if privateKey, err := getRSAPrivateKey([]byte(pem)); err != nil {
		return nil, err
	} else {
		return &JWTUtil{
			privateKey: privateKey,
			config:     config,
		}, nil
	}
}

// JWTClaims store all jwt claims
type JWTClaims map[string]interface{}

func getRSAPrivateKey(pemPrivateKey []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemPrivateKey)
	if block == nil {
		return nil, errors.New("no PEM data found in the input bytes")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// GenerateJWT create new jwe message (sign and encrypt)
func (j *JWTUtil) GenerateJWT(claims JWTClaims) (string, error) {
	t := jwt.New()
	for k, v := range claims {
		if err := t.Set(k, v); err != nil {
			return "", err
		}
	}

	signed, err := jwt.Sign(t, j.config.SignatureAlgo, j.privateKey)
	if err != nil {
		return "", err
	}

	if !j.config.EnableJWE {
		return string(signed), nil
	}

	encrypted, err := jwe.Encrypt(
		signed,
		j.config.KeyEncryptAlgo,
		&j.privateKey.PublicKey,
		j.config.ContentEncryptAlgo,
		j.config.JWECompressAlgo)
	if err != nil {
		return "", err
	}

	return string(encrypted), nil
}

// VerifyJWT decrypt and verify string jwe message in token
func (j *JWTUtil) VerifyJWT(token string) (jwt.Token, error) {
	bytesToken := []byte(token)
	if j.config.EnableJWE {
		decrypted, err := jwe.Decrypt([]byte(token), j.config.KeyEncryptAlgo, j.privateKey)
		if err != nil {
			return nil, err
		}
		bytesToken = decrypted
	}

	t, err := jwt.Parse(bytesToken, jwt.WithVerify(j.config.SignatureAlgo, j.privateKey.PublicKey))
	if err != nil {
		return nil, err
	}

	err = jwt.Validate(t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// GenerateRSAPrivateKey create new rsa private key with given bits length
// and return it in PEM format
func GenerateRSAPrivateKey(bits int) ([]byte, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}

	pkcs1PrivateKey := x509.MarshalPKCS1PrivateKey(privKey)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pkcs1PrivateKey,
	}), nil
}
