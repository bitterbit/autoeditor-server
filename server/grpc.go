package server

import (
	"context"
	"galtashma/editor-server/app"
	"log"
	"net"

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
	// Extract the code and prompt from the request
	code := req.GetCode()
	prompt := req.GetPrompt()

	// Call the OpenAI GPT model to modify the code based on the prompt
	modifiedCode, err := s.openaiSession.ModifyCode(ctx, code, prompt)
	if err != nil {
		return nil, err
	}

	// Create and return the RPC response with the modified code
	res := &editorv1.CodeModificationResponse{
		ModifiedCode: modifiedCode,
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
