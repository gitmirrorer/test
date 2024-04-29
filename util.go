package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"net"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

func GeneratePrivateRSAKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func CreatePrivateSSHKey(bitSize int) (privateKeyInPEM, authorizedKeys []byte, err error) {
	privateKey, err := GeneratePrivateRSAKey(bitSize)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key")
	}

	privateKeyInPEM = pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	})

	publicSSHKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to parse public key")
	}
	return privateKeyInPEM, ssh.MarshalAuthorizedKey(publicSSHKey), nil
}

func GetPublicKeyForServer(host string) (ssh.PublicKey, error) {
	var hostKey ssh.PublicKey
	config := &ssh.ClientConfig{
		User: "git",
		Auth: nil,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			hostKey = key
			return io.EOF
		},
	}
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		if hostKey != nil {
			return hostKey, nil
		}
		return nil, errors.Wrap(err, "failure during connect")
	}
	_ = client.Close()
	return nil, errors.New("expected error during connect")
}

func CreateBasicAuthHeaderValue(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}
