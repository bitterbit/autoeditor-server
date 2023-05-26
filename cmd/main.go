package main

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"buf.build/gen/go/galtashma/editor/grpc/go/editor/editorgrpc"
	editorv1 "buf.build/gen/go/galtashma/editor/protocolbuffers/go/editor"
)

type gitServer struct{}

func (s *gitServer) GetTrackedFiles(ctx context.Context, empty *editorv1.Empty) (*editorv1.FileList, error) {
	files, err := getGitTrackedFiles()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tracked files")
	}

	return &editorv1.FileList{Files: files}, nil
}

func (s *gitServer) GetFileDetails(ctx context.Context, request *editorv1.FileRequest) (*editorv1.FileDetails, error) {
	content, err := getFileContent(request.Filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file content")
	}

	changes, err := getGitChanges(request.Filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get git changes")
	}

	return &editorv1.FileDetails{
		Content: content,
		Changes: changes,
	}, nil
}

func getGitTrackedFiles() ([]string, error) {
	repo, err := git.PlainOpen(getCurrentDirectory())
	if err != nil {
		return nil, errors.Wrap(err, "failed to open git repository")
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get git worktree")
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get git status")
	}

	files := make([]string, 0)
	for file, fs := range status {
		if fs.Worktree == git.Unmodified {
			files = append(files, file)
		}
	}

	return files, nil
}

func getFileContent(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", errors.Wrap(err, "failed to read file content")
	}

	return string(content), nil
}

func getGitChanges(filename string) (string, error) {
	repo, err := git.PlainOpen(getCurrentDirectory())
	if err != nil {
		return "", errors.Wrap(err, "failed to open git repository")
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
	grpcServer := grpc.NewServer()
	editorgrpc.RegisterGitServiceServer(grpcServer, &gitServer{})

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
