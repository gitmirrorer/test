package test

import (
	"context"
	"encoding/base64"
	"fmt"
	"hash/crc32"

	"code.gitea.io/sdk/gitea"
	"github.com/pkg/errors"
)

type Session struct {
	client   *gitea.Client
	Username string
}

func (s *Session) CreateRepo(ctx context.Context, repoName string) error {
	s.client.SetContext(ctx)
	_, _, err := s.client.CreateRepo(gitea.CreateRepoOption{
		Name:          repoName,
		Description:   "",
		Private:       true,
		IssueLabels:   "",
		AutoInit:      false,
		Template:      false,
		Gitignores:    "",
		License:       "",
		Readme:        "",
		DefaultBranch: "",
		TrustModel:    "",
	})
	if err != nil {
		return errors.Wrap(err, "unable to create new repo")
	}
	return nil
}

func (s *Session) CreateFile(ctx context.Context, repoName, fileName, contents string) error {
	s.client.SetContext(ctx)
	_, _, err := s.client.CreateFile(s.Username, repoName, fileName, gitea.CreateFileOptions{
		FileOptions: gitea.FileOptions{
			Message: "added " + fileName,
		},
		Content: base64.StdEncoding.EncodeToString([]byte(contents)),
	})
	if err != nil {
		return errors.Wrap(err, "unable to create new file")
	}
	return nil
}

func (s *Session) CreateTag(ctx context.Context, repoName, tagName string) error {
	s.client.SetContext(ctx)

	repo, _, err := s.client.GetRepo(s.Username, repoName)
	if err != nil {
		return errors.Wrap(err, "unable to get repo")
	}

	_, _, err = s.client.CreateTag(s.Username, repoName, gitea.CreateTagOption{
		TagName: tagName,
		Message: "",
		Target:  repo.DefaultBranch,
	})
	if err != nil {
		return errors.Wrap(err, "unable to create new tag")
	}
	return nil
}

func (s *Session) GetRefs(ctx context.Context, repoName string) (map[string]string, error) {
	s.client.SetContext(ctx)
	refs, _, err := s.client.GetRepoRefs(s.Username, repoName, "")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create new repo")
	}
	result := make(map[string]string)
	for _, ref := range refs {
		if ref.Object == nil {
			continue
		}
		result[ref.Ref] = ref.Object.SHA
	}
	return result, nil
}

func (s *Session) AddPublicKey(ctx context.Context, publicKeyBytes []byte) error {
	s.client.SetContext(ctx)

	_, _, err := s.client.CreatePublicKey(gitea.CreateKeyOption{
		Title:    fmt.Sprintf("publickey-%d", crc32.ChecksumIEEE(publicKeyBytes)),
		Key:      string(publicKeyBytes),
		ReadOnly: false,
	})
	if err != nil {
		return errors.Wrap(err, "unable to create public key")
	}
	return nil
}

func (s *Session) RemoveAllPublicKeys(ctx context.Context) error {
	s.client.SetContext(ctx)

	for {
		publicKeys, _, err := s.client.ListMyPublicKeys(gitea.ListPublicKeysOptions{})
		if err != nil {
			return errors.Wrap(err, "unable to list public keys")
		}
		if len(publicKeys) == 0 {
			return nil
		}

		for _, key := range publicKeys {
			_, err := s.client.DeletePublicKey(key.ID)
			if err != nil {
				return errors.Wrap(err, "unable to delete public key")
			}
		}
	}
}
