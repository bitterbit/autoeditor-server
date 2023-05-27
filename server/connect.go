package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bufbuild/connect-go"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	editorv1connect "buf.build/gen/go/galtashma/editor/bufbuild/connect-go/editor/editorconnect"
	editorv1 "buf.build/gen/go/galtashma/editor/protocolbuffers/go/editor"
)

type ConnectServer struct {
	grpcServer *GRPCServer
}

func (s *ConnectServer) GetTrackedFiles(ctx context.Context, req *connect.Request[editorv1.Empty]) (*connect.Response[editorv1.FileList], error) {
	fileList, err := s.grpcServer.GetTrackedFiles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(fileList), nil
}

func (s *ConnectServer) GetFileDetails(ctx context.Context, req *connect.Request[editorv1.FileRequest]) (*connect.Response[editorv1.FileDetails], error) {
	fileDetails, err := s.grpcServer.GetFileDetails(ctx, req.Msg)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(fileDetails), nil
}

func (s *ConnectServer) ModifyCode(ctx context.Context, req *connect.Request[editorv1.CodeModificationRequest]) (*connect.Response[editorv1.CodeModificationResponse], error) {
	codeModificationResponse, err := s.grpcServer.ModifyCode(ctx, req.Msg)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(codeModificationResponse), nil
}

func StartConnectServer(directory, addr, openapiKey string) {
	server := &ConnectServer{
		grpcServer: NewGRPCServer(directory, openapiKey),
	}

	router := http.NewServeMux()
	path, handler := editorv1connect.NewGitServiceHandler(server)
	router.Handle(path, handler)

	fmt.Println("Starting Connect server on", addr, "with handler at", path)
	http.ListenAndServe(
		"localhost:8080",
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(router, &http2.Server{}),
	)
}
