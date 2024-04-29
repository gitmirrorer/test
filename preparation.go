package test

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/slices"
)

type RepositoryConfig struct {
	RepoName string
	HTTP     HTTPConfig
	SSH      SSHConfig
	Session  *Session
}

type HTTPConfig struct {
	URL    string
	Header map[string]string
}

type SSHConfig struct {
	URL                string
	Username           string
	PrivateKey         []byte
	PrivateKeyPassword string
	KnownHosts         []ssh.PublicKey
}

type Preparation struct {
	Source      RepositoryConfig
	Destination RepositoryConfig
	Server      *Server
}

func (p *Preparation) Close(context.Context) error {
	return nil
}

func createUserName(prefix, testName, suffix string) string {
	validRunes := []rune("abcdefghijklmnopqrstuvwxyz-_0123467890")

	r := []rune(prefix + strings.ToLower(testName) + suffix)

	for i := len(r) - 1; i >= 0; i-- {
		if !slices.Contains(validRunes, r[i]) {
			r[i] = '_'
		}
	}
	return strings.TrimFunc(string(r), func(r rune) bool {
		return r == '_' || r == '-'
	})
}

func PrepareRepositoriesOnAServer(
	ctx context.Context,
	server *Server,
	testName string,
	sourceFiles,
	destinationFiles map[string]string) (*Preparation, error) {
	userNameSource := createUserName("", testName, "-src")
	passwordSource := "src"
	userNameDestination := createUserName("", testName, "-dst")
	passwordDestination := "dst"

	// create publickey hostkey from gitea
	publicKey, err := GetPublicKeyForServer(server.SSHAddress())
	if err != nil {
		return nil, errors.Wrap(err, "unable to get public keys")
	}

	prep := Preparation{
		Source: RepositoryConfig{
			RepoName: "src",
			HTTP: HTTPConfig{
				URL: "http://" + server.HTTPAddress() + "/" + userNameSource + "/src.git", //nolint: goconst // allow http scheme
				Header: map[string]string{
					"Authorization": CreateBasicAuthHeaderValue(userNameSource, passwordSource),
				},
			},
			SSH: SSHConfig{
				URL:                "ssh://" + server.SSHAddress() + "/" + userNameSource + "/src.git",
				Username:           "git",
				PrivateKey:         nil, // set later
				PrivateKeyPassword: "",
				KnownHosts:         []ssh.PublicKey{publicKey},
			},
		},
		Destination: RepositoryConfig{
			RepoName: "dst",
			HTTP: HTTPConfig{
				URL: "http://" + server.HTTPAddress() + "/" + userNameDestination + "/dst.git",
				Header: map[string]string{
					"Authorization": CreateBasicAuthHeaderValue(userNameDestination, passwordDestination),
				},
			},
			SSH: SSHConfig{
				URL:                "ssh://" + server.SSHAddress() + "/" + userNameDestination + "/dst.git",
				Username:           "git",
				PrivateKey:         nil, // set later
				PrivateKeyPassword: "",
				KnownHosts:         []ssh.PublicKey{publicKey},
			},
		},
		Server: server,
	}

	// create users
	if err := server.CreateUser(
		ctx,
		userNameSource,
		passwordSource,
		userNameSource+"@example.com",
	); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := server.CreateUser(
		ctx,
		userNameDestination,
		passwordDestination,
		userNameDestination+"@example.com",
	); err != nil {
		return nil, errors.WithStack(err)
	}

	// create sessions
	prep.Source.Session, err = server.NewSession(userNameSource, passwordSource)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	prep.Destination.Session, err = server.NewSession(userNameDestination, passwordDestination)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	const privateKeyBitSize = 2047

	sourcePrivateKey, sourcePublicKey, err := CreatePrivateSSHKey(privateKeyBitSize)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = prep.Source.Session.AddPublicKey(ctx, sourcePublicKey); err != nil {
		return nil, errors.WithStack(err)
	}
	prep.Source.SSH.PrivateKey = sourcePrivateKey

	destinationPrivateKey, destinationPublicKey, err := CreatePrivateSSHKey(privateKeyBitSize)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err = prep.Destination.Session.AddPublicKey(ctx, destinationPublicKey); err != nil {
		return nil, errors.WithStack(err)
	}
	prep.Destination.SSH.PrivateKey = destinationPrivateKey

	// create repos
	if err := prep.Source.Session.CreateRepo(ctx, prep.Source.RepoName); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := prep.Destination.Session.CreateRepo(ctx, prep.Destination.RepoName); err != nil {
		return nil, errors.WithStack(err)
	}

	// create first commits to the repos
	for file, contents := range sourceFiles {
		if err := prep.Source.Session.CreateFile(ctx, prep.Source.RepoName, file, contents); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	for file, contents := range destinationFiles {
		if err := prep.Destination.Session.CreateFile(ctx, prep.Destination.RepoName, file, contents); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	return &prep, nil
}
