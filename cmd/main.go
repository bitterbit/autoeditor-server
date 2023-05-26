package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"buf.build/gen/go/galtashma/editor/grpc/go/editor/editorgrpc"
	editorv1 "buf.build/gen/go/galtashma/editor/protocolbuffers/go/editor"
)

type gitServer struct {
	rootDirectory string
}

func (s *gitServer) GetTrackedFiles(ctx context.Context, empty *editorv1.Empty) (*editorv1.FileList, error) {
	files, err := getGitTrackedFiles(s.rootDirectory)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tracked files")
	}

	return &editorv1.FileList{Files: files}, nil
}

func (s *gitServer) GetFileDetails(ctx context.Context, request *editorv1.FileRequest) (*editorv1.FileDetails, error) {
	content, err := getFileContent(s.rootDirectory, request.Filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file content")
	}

	changes, err := getGitChanges(s.rootDirectory, request.Filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get git changes")
	}

	return &editorv1.FileDetails{
		Content: content,
		Changes: changes,
	}, nil
}

func getGitTrackedFiles(directory string) ([]string, error) {
	repo, err := git.PlainOpen(directory)
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
func getFileContent(rootDirectory, filePath string) (string, error) {
	repo, err := git.PlainOpen(rootDirectory)
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

func getGitChanges(directory, filename string) (string, error) {
	repo, err := git.PlainOpen(directory)

	if err != nil {
		return "", errors.Wrapf(err, "failed to open git repository %v", directory)
	}

	commit, err := repo.Head()
	if err != nil {
		return "", errors.Wrap(err, "failed to get git head commit")
	}

	commitObj, err := repo.CommitObject(commit.Hash())
	if err != nil {
		return "", errors.Wrap(err, "failed to get git commit object")
	}

	diff, err := commitObj.PatchContext(context.Background(), commitObj)
	if err != nil {
		return "", errors.Wrap(err, "failed to get git diff")
	}

	for _, patches := range diff.FilePatches() {
		from, to := patches.Files()
		if from.Path() == filename || to.Path() == filename {
			// TODO
			return "", nil
		}
	}

	return "", nil
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func main() {
	rootDirectory := flag.String("root", getCurrentDirectory(), "Root Git directory")
	flag.Parse()

	grpcServer := grpc.NewServer()

	log.Println("root directory", *rootDirectory)
	editorgrpc.RegisterGitServiceServer(grpcServer, &gitServer{
		rootDirectory: *rootDirectory,
	})

	log.Println("Starting gRPC server on port 50051...")
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
