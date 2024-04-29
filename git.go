package test

import (
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	"github.com/go-git/go-git/v5/plumbing/transport/file"
	"github.com/pkg/errors"
)

type gitImpl struct {
}

var Git gitImpl

func (gitImpl) AddFileAndCommit(dir, fileName, fileContents string) error {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return errors.Wrap(err, "unable to open repo")
	}

	worktree, err := repo.Worktree()
	if err != nil {
		if errors.Is(err, git.ErrIsBareRepository) {
			return Git.AddfileAndCommitInBareRepo(dir, fileName, fileContents)
		}

		return errors.Wrap(err, "unable to get worktree")
	}

	f, err := worktree.Filesystem.Create(fileName)
	if err != nil {
		return errors.Wrap(err, "unable to create file on fs")
	}

	if _, err := f.Write([]byte(fileContents)); err != nil {
		return errors.Wrap(err, "unable to write file on fs")
	}
	if err := f.Close(); err != nil {
		return errors.Wrap(err, "unable to close file on fs")
	}

	_, err = worktree.Add(fileName)
	if err != nil {
		return errors.Wrap(err, "unable to add file")
	}

	commitHash, err := worktree.Commit("adding "+fileName, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "John Doe",
			Email: "john@doe.org",
			When:  time.Now(),
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to commit file")
	}

	_, err = repo.CommitObject(commitHash)
	if err != nil {
		return errors.Wrap(err, "unable to commit object")
	}
	return nil
}

func (gitImpl) AddfileAndCommitInBareRepo(dir, fileName, fileContents string) error {
	tmpRepo, err := Git.CreateRepo(false)
	if err != nil {
		return errors.Wrap(err, "unable to create repo")
	}
	defer os.RemoveAll(tmpRepo)

	repo, err := git.PlainOpen(tmpRepo)
	if err != nil {
		return errors.WithStack(err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return errors.WithStack(err)
	}

	client.InstallProtocol("file", file.DefaultClient)

	remote, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"file://" + dir},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	err = wt.Pull(&git.PullOptions{
		RemoteName: remote.Config().Name,
		RemoteURL:  remote.Config().URLs[0],
	})
	if err != nil {
		if !errors.Is(err, transport.ErrEmptyRemoteRepository) {
			return errors.WithStack(err)
		}
	}

	if err := Git.AddFileAndCommit(tmpRepo, fileName, fileContents); err != nil {
		return errors.WithStack(err)
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: remote.Config().Name,
		RemoteURL:  remote.Config().URLs[0],
		RefSpecs: []config.RefSpec{
			"+refs/heads/master:refs/heads/master",
		},
		Auth: nil,
	})
	if err != nil {
		return errors.Wrap(err, "unable to push")
	}
	return nil
}

func (gitImpl) CreateRepo(bare bool) (string, error) {
	tempDir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temp dir")
	}
	repo, err := git.PlainInit(tempDir, bare)
	if err != nil {
		return "", errors.Wrap(err, "unable to init git")
	}

	cfg, err := repo.Config()
	if err != nil {
		return "", errors.Wrap(err, "unable to get config")
	}

	if err := repo.SetConfig(cfg); err != nil {
		return "", errors.Wrap(err, "unable to set config")
	}

	return tempDir, nil
}

func (gitImpl) GetRefs(dir string) (map[string]string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open repo")
	}

	iter, err := repo.References()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get log")
	}
	refs := make(map[string]string)
	err = iter.ForEach(func(reference *plumbing.Reference) error {
		refs[reference.Name().String()] = reference.Hash().String()
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to loop trough refs")
	}
	return refs, nil
}

func (gitImpl) CreateBranch(dir, branchName string) error {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return errors.Wrap(err, "unable to open repo")
	}

	err = repo.CreateBranch(&config.Branch{
		Name:  branchName,
		Merge: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		return errors.Wrap(err, "unable to create branch")
	}
	return nil
}

func (gitImpl) CheckoutBranch(dir, branchName string) error {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return errors.Wrap(err, "unable to open repo")
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "unable to get worktree")
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Hash:                      plumbing.ZeroHash,
		Branch:                    plumbing.NewBranchReferenceName(branchName),
		Create:                    true,
		Force:                     false,
		Keep:                      false,
		SparseCheckoutDirectories: nil,
	}); err != nil {
		return errors.Wrap(err, "unable to checkout branch")
	}
	return nil
}
