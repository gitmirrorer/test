package test

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"unicode"

	"code.gitea.io/sdk/gitea"
	"github.com/docker/docker/api/types/container"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Server struct {
	container testcontainers.Container
	SSHHost   string
	SSHPort   int
	HTTPHost  string
	HTTPPort  int
}

func (s *Server) SSHAddress() string {
	return net.JoinHostPort(s.SSHHost, strconv.Itoa(s.SSHPort))
}

func (s *Server) HTTPAddress() string {
	return net.JoinHostPort(s.HTTPHost, strconv.Itoa(s.HTTPPort))
}

func ListenAndServe(ctx context.Context, giteaImage string) (*Server, error) {
	req := testcontainers.ContainerRequest{
		Image:        cmp.Or(giteaImage, "gitea/gitea:1.19.0"),
		Name:         fmt.Sprintf("test-gitea-%s", uuid.NewString()),
		ExposedPorts: []string{"3000/tcp", "22/tcp"},
		Env: map[string]string{
			"GITEA__security__INSTALL_LOCK": "true",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("3000/tcp"),
			wait.ForListeningPort("22/tcp"),
			wait.ForHTTP("/api/healthz").WithPort("3000/tcp").
				WithResponseMatcher(func(body io.Reader) bool {
					var response struct {
						Status string `json:"status"`
					}
					if err := json.NewDecoder(body).Decode(&response); err != nil {
						return false
					}
					return response.Status == "pass"
				}),
		),
		HostConfigModifier: func(config *container.HostConfig) {
			config.AutoRemove = true
		},
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to start container")
	}

	_, _, err = c.Exec(ctx, []string{
		"su", "git", "bash", "-c", "gitea migrate",
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to run gitea migrate")
	}

	s := Server{container: c}

	ip, err := c.Host(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get container ip")
	}

	port, err := c.MappedPort(ctx, "22/tcp")
	if err != nil {
		return nil, errors.Wrap(err, "unable to get port of 22/tcp")
	}
	s.SSHHost = ip
	s.SSHPort = port.Int()

	port, err = c.MappedPort(ctx, "3000/tcp")
	if err != nil {
		return nil, errors.Wrap(err, "unable to get port of 3000/tcp")
	}
	s.HTTPHost = ip
	s.HTTPPort = port.Int()

	return &s, nil
}

func (s *Server) Close(ctx context.Context) error {
	return s.container.Terminate(ctx)
}

func (s *Server) CreateUser(ctx context.Context, username, password, email string) error {
	_, r, err := s.container.Exec(ctx, []string{
		"su", "git", "bash", "-c", "gitea admin user create --username " + username +
			" --password " + password + " --email " + email + " --must-change-password=false",
	})
	if err != nil {
		return errors.Wrap(err, "unable to run gitea admin user create")
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "unable get response")
	}
	response := string(bytes.TrimFunc(buf, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		return r == 0 || r == 1
	}))

	if !strings.HasSuffix(response, "New user '"+username+"' has been successfully created!") {
		return errors.Errorf("unable to create new user, response was: %q", response)
	}
	return nil
}

func (s *Server) NewSession(username, password string) (*Session, error) {
	client, err := gitea.NewClient(
		"http://"+s.HTTPAddress(),
		gitea.SetBasicAuth(username, password),
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Session{
		client:   client,
		Username: username,
	}, nil
}
