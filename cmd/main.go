package main

import (
	"context"
	"flag"
	"galtashma/editor-server/app"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"buf.build/gen/go/galtashma/editor/grpc/go/editor/editorgrpc"
	editorv1 "buf.build/gen/go/galtashma/editor/protocolbuffers/go/editor"
)

type Server struct {
	git *app.Git
}

// GetTrackedFiles retrieves the list of tracked files within a Git repository.
func (s *Server) GetTrackedFiles(ctx context.Context, empty *editorv1.Empty) (*editorv1.FileList, error) {
	files, err := s.git.GetGitTrackedFiles()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tracked files")
	}

	return &editorv1.FileList{Files: files}, nil
}

// GetFileDetails retrieves the details of a file at the specified path within a Git repository.
func (s *Server) GetFileDetails(ctx context.Context, request *editorv1.FileRequest) (*editorv1.FileDetails, error) {
	content, err := s.git.GetFileContent(request.Filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file content")
	}

	state, err := s.git.GetFileState(request.Filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file state")
	}

	originalContent := ""
	if state != editorv1.FileState_UNMODIFIED {
		originalContent, err = s.git.GetFileContentsHEAD(request.Filename)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get git changes")
		}
	} else {
		originalContent = content
	}

	return &editorv1.FileDetails{
		Content:  content,
		Original: originalContent,
		State:    state,
	}, nil
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func NewServer(cwd string) *Server {
	return &Server{
		git: app.NewGit(cwd),
	}
}

func main() {
	rootDirectory := flag.String("root", getCurrentDirectory(), "Root Git directory")
	flag.Parse()

	grpcServer := grpc.NewServer()

	log.Println("root directory", *rootDirectory)
	editorgrpc.RegisterGitServiceServer(grpcServer, NewServer(*rootDirectory))

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
