package server

import (
	"context"
	"fmt"
	"galtashma/editor-server/app"
	"log"
	"net"
	"path/filepath"
	"strings"

	"buf.build/gen/go/galtashma/editor/grpc/go/editor/editorgrpc"
	editorv1 "buf.build/gen/go/galtashma/editor/protocolbuffers/go/editor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/pkg/errors"
)

type GRPCServer struct {
	git           *app.Git
	openaiSession *app.OpenAISession
}

func NewGRPCServer(rootDirectory, openapiKey string) *GRPCServer {
	return &GRPCServer{
		git:           app.NewGit(rootDirectory),
		openaiSession: app.NewOpenAISession(openapiKey),
	}
}

// GetTrackedFiles retrieves the list of tracked files within a Git repository.
func (s *GRPCServer) GetTrackedFiles(ctx context.Context, empty *editorv1.Empty) (*editorv1.FileList, error) {
	files, err := s.git.GetGitTrackedFiles()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tracked files")
	}

	return &editorv1.FileList{Files: files}, nil
}

// GetFileDetails retrieves the details of a file at the specified path within a Git repository.
func (s *GRPCServer) GetFileDetails(ctx context.Context, request *editorv1.FileRequest) (*editorv1.FileDetails, error) {
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

func (s *GRPCServer) ModifyCode(ctx context.Context, req *editorv1.CodeModificationRequest) (*editorv1.CodeModificationResponse, error) {
	prompt := req.GetPrompt()

	lang := filepath.Ext(req.GetPath())

	content, err := s.git.GetFileContent(req.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file content")
	}

	start := req.GetLineStart()
	if start < 0 {
		start = 0
	}
	end := req.GetLineEnd()
	if end < 0 {
		end = -1
	}

	lines := strings.Split(content, "\n")
	code := strings.Join(lines[start:end], "\n")

	// Call the OpenAI GPT model to modify the code based on the prompt
	modifiedCode, err := s.openaiSession.ModifyCode(ctx, code, prompt, lang)
	if err != nil {
		return nil, err
	}

	explenation, err := s.openaiSession.ExplainModification(ctx, prompt, modifiedCode)
	if err != nil {
		return nil, err
	}

	fmt.Println("Modification", modifiedCode)

	// Create and return the RPC response with the modified code
	res := &editorv1.CodeModificationResponse{
		Explenation:   explenation,
		ModifiedFiles: []string{req.GetPath()},
	}

	return res, nil
}

func StartGRPCServer(rootDirectory, addr, openaiKey string) error {
	grpcServer := grpc.NewServer()

	log.Println("root directory", rootDirectory)
	editorgrpc.RegisterGitServiceServer(grpcServer, NewGRPCServer(rootDirectory, openaiKey))

	log.Printf("Starting gRPC server on %s \n", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}

	return nil
}
