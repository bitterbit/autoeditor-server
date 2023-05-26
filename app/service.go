package app

import (
	"bytes"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	editorv1 "buf.build/gen/go/galtashma/editor/protocolbuffers/go/editor"
)

type Git struct {
	rootDirectory string
}

func NewGit(rootDirectory string) *Git {
	return &Git{rootDirectory: rootDirectory}
}

func (s *Git) GetFileState(filePath string) (editorv1.FileState, error) {
	repo, err := git.PlainOpen(s.rootDirectory)
	if err != nil {
		return editorv1.FileState_UNMODIFIED, errors.Wrap(err, "failed to open git repository")
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return editorv1.FileState_UNMODIFIED, err
	}

	status, err := worktree.Status()
	if err != nil {
		return editorv1.FileState_UNMODIFIED, err
	}

	switch status.File(filePath).Worktree {
	case git.Added:
		return editorv1.FileState_ADDED, nil
	case git.Copied:
		return editorv1.FileState_COPIED, nil
	case git.Deleted:
		return editorv1.FileState_DELETED, nil
	case git.Modified:
		return editorv1.FileState_MODIFIED, nil
	case git.Renamed:
		return editorv1.FileState_RENAMED, nil
	case git.UpdatedButUnmerged:
		return editorv1.FileState_UPDATED_BUT_UNMERGED, nil
	default:
		return editorv1.FileState_UNMODIFIED, nil
	}
}

func (s *Git) GetGitTrackedFiles() ([]string, error) {
	repo, err := git.PlainOpen(s.rootDirectory)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open git repository")
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	filesystem := worktree.Filesystem
	files := getAllFiles(filesystem, "/", nil)

	gitignoreMatcher := gitignore.NewMatcher(worktree.Excludes)

	files = lo.Filter(files, func(path string, index int) bool {
		if strings.HasPrefix(path, ".git/") {
			return false
		}
		return !gitignoreMatcher.Match([]string{path}, false)
	})

	return files, nil
}

// getAllFiles recursively retrieves all file paths under the specified directory using the given filesystem.
// The currentDirectory parameter represents the current directory being traversed, and the item parameter represents
// the current file or directory being processed. If item is nil, the function starts traversing from the specified root directory.
// The function returns a list of all file paths found.
func getAllFiles(filesystem billy.Filesystem, currentDirectory string, item fs.FileInfo) []string {
	if item == nil {
		files, _ := filesystem.ReadDir(currentDirectory)
		allFiles := lo.Map(files, func(file fs.FileInfo, index int) []string {
			return getAllFiles(filesystem, currentDirectory, file)
		})

		return lo.Flatten(allFiles)
	}

	var path string
	if currentDirectory == "/" {
		path = item.Name()
	} else {
		path = filepath.Join(currentDirectory, item.Name())
	}

	if item.IsDir() {
		files, err := filesystem.ReadDir(path)
		if err != nil {
			panic(err)
		}

		allFiles := lo.Map(files, func(file fs.FileInfo, index int) []string {
			return getAllFiles(filesystem, path, file)
		})

		return lo.Flatten(allFiles)
	}

	return []string{path}
}

// getFileContent retrieves the contents of a file at the specified path within a Git repository.
// It returns the file content as a string and any error encountered during the process.
func (s *Git) GetFileContent(filePath string) (string, error) {
	repo, err := git.PlainOpen(s.rootDirectory)
	if err != nil {
		return "", errors.Wrap(err, "failed to open git repository")
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	file, err := worktree.Filesystem.Open(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to open file: %s", filePath)
	}
	defer file.Close()

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, file)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file: %s", filePath)
	}

	return buf.String(), nil
}

func (s *Git) GetFileContentsHEAD(filePath string) (string, error) {
	repo, err := git.PlainOpen(s.rootDirectory)
	if err != nil {
		return "", errors.Wrapf(err, "failed to open git repository %s", s.rootDirectory)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", errors.Wrap(err, "failed to get HEAD reference")
	}

	// Get the commit object from the HEAD reference
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", errors.Wrap(err, "failed to get commit object")
	}

	// Get the file tree from the commit
	tree, err := commit.Tree()
	if err != nil {
		return "", errors.Wrap(err, "failed to get commit tree")
	}

	// Get the file entry from the tree
	fileEntry, err := tree.FindEntry(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to find file entry for path: %s", filePath)
	}

	blob, err := repo.BlobObject(fileEntry.Hash)
	if err != nil {
		return "", errors.Wrap(err, "failed to get blob object")
	}

	reader, err := blob.Reader()
	if err != nil {
		return "", errors.Wrap(err, "failed to read blob file")
	}

	buffer := bytes.NewBuffer(nil)
	io.Copy(buffer, reader)

	return buffer.String(), nil
}
