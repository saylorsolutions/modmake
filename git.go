package modmake

import (
	"errors"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gitcache "github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/saylorsolutions/cache"
	"os"
	"path/filepath"
	"sync"
)

type GitTools struct {
	mux     sync.RWMutex
	rootDir *cache.Value[string]
	head    *cache.Value[*plumbing.Reference]
}

// NewGitTools returns a new instance of GitTools that may be reused for multiple operations.
// A single instance should only be used in a single repository or submodule.
// If an error occurs while trying to cache the Git context, then this function will panic.
func NewGitTools() *GitTools {
	return initGitInst()
}

func initGitInst() *GitTools {
	tools := new(GitTools)
	tools.rootDir = cache.New[string](func() (string, error) {
		tools.mux.RLock()
		defer tools.mux.RUnlock()
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		var (
			prev = ""
		)
		for cwd != prev {
			fd, err := os.Stat(filepath.Join(cwd, ".git"))
			if err == nil && fd.IsDir() {
				return cwd, nil
			}
			prev = cwd
			cwd = filepath.Dir(cwd)
		}
		return "", errors.New("unable to find the git repository root directory")
	})
	tools.head = cache.New(func() (*plumbing.Reference, error) {
		tools.mux.RLock()
		defer tools.mux.RUnlock()
		root, err := tools.rootDir.Get()
		if err != nil {
			return nil, err
		}
		fs := osfs.New(filepath.Join(root, ".git"))
		s := filesystem.NewStorageWithOptions(fs, gitcache.NewObjectLRUDefault(), filesystem.Options{
			KeepDescriptors: true,
		})
		r, err := git.Open(s, fs)
		if err != nil {
			return nil, err
		}
		ref, err := r.Head()
		if err != nil {
			return nil, err
		}
		return ref, nil
	})
	return tools
}

func (g *GitTools) InvalidateCache() {
	g.mux.Lock()
	defer g.mux.Unlock()
	g.rootDir.Invalidate()
	g.head.Invalidate()
}

func (g *GitTools) RepositoryRoot() string {
	return g.rootDir.MustGet()
}

func (g *GitTools) BranchName() string {
	return g.head.MustGet().String()
}

func (g *GitTools) CommitHash() string {
	return g.head.MustGet().Hash().String()
}
