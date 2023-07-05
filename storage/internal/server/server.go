package server

import (
	// buildin
	"context"
	"io"
	"log"
	"net"
	"strconv"

	// other
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	// local
	pb "github.com/muskelo/ns_server/protos/storage"
	"github.com/muskelo/ns_server/storage/internal/filemanager"
)

// run server with default grpc server
func Serve(addr string, server *Server) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			UnaryLogger(),
		),
		grpc.ChainStreamInterceptor(
			StreamLogger(),
		),
	)
	pb.RegisterStorageServiceServer(s, server)
	reflection.Register(s)
	return s.Serve(lis)
}

func New(fm *filemanager.FileManager) *Server {
	return &Server{
		FM: fm,
	}
}

type Server struct {
	FM *filemanager.FileManager
}

func (s *Server) Mkdir(ctx context.Context, request *pb.MkdirRequest) (*pb.MkdirResponse, error) {
	exist, err := s.FM.IsExist(request.Path)
	if err != nil {
		return nil, err
	}
	if exist {
		return nil, status.Errorf(codes.AlreadyExists, "Directory of file %v already exist", request.Path)
	}

	err = s.FM.Mkdir(request.Path)
	return &pb.MkdirResponse{}, err
}

func (s *Server) ReadDir(ctx context.Context, request *pb.ReadDirRequest) (*pb.ReadDirResponse, error) {
	exist, err := s.FM.IsDirExist(request.Path)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, status.Errorf(codes.NotFound, "Directory %v not exist", request.GetPath())
	}

	files, dirs, err := s.FM.ReadDir(request.GetPath())
	if err != nil {
		return nil, err
	}

	response := &pb.ReadDirResponse{
		Files: make([]*pb.ReadDirResponse_File, 0),
		Dirs:  make([]*pb.ReadDirResponse_Dir, 0),
	}
	for _, file := range files {
		response.Files = append(response.Files, &pb.ReadDirResponse_File{
			Name: file.Name,
			Path: file.Path,
		})
	}
	for _, dir := range dirs {
		response.Dirs = append(response.Dirs, &pb.ReadDirResponse_Dir{
			Name: dir.Name,
			Path: dir.Path,
		})
	}
	return response, nil
}

func (s *Server) Remove(ctx context.Context, request *pb.RemoveRequest) (*pb.RemoveResponse, error) {
	// handle file
	exist, err := s.FM.IsFileExist(request.Path)
	if err != nil {
		return nil, err
	}
	if exist {
		return &pb.RemoveResponse{}, s.FM.Remove(request.Path)
	}

	// handle Directory
	exist, err = s.FM.IsDirExist(request.Path)
	if err != nil {
		return nil, err
	}
	if exist {
		files, dirs, err := s.FM.ReadDir(request.Path)
		if err != nil {
			return nil, err
		}
		if len(files) > 0 || len(dirs) > 0 {
			return nil, status.Error(codes.FailedPrecondition, "Directory not empty")
		}
		return &pb.RemoveResponse{}, s.FM.Remove(request.Path)
	}

	return nil, status.Errorf(codes.NotFound, "File or Directory %v not found", request.Path)
}

func (s *Server) parseDownloadMD(stream pb.StorageService_DownloadServer) (path string) {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return
	}
	v := md.Get("path")
	if len(v) > 0 {
		path = v[0]
	}
	return
}
func (s *Server) Download(request *pb.DownloadRequest, stream pb.StorageService_DownloadServer) error {
	path := s.parseDownloadMD(stream)
	if path == "" {
		return status.Error(codes.InvalidArgument, "missing path")
	}

	info, exist, err := s.FM.Stat(path)
	if err != nil {
		return err
	}
	if !exist {
		return status.Errorf(codes.NotFound, "file %v not found", path)
	}

	md := metadata.Pairs("name", info.Name(), "size", strconv.FormatInt(info.Size(), 10))
	if err := stream.SendHeader(md); err != nil {
		return err
	}

	file, err := s.FM.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	streamWriter := new(pb.StreamWriter)
	streamWriter.StorageService_DownloadServer(stream)
	_, err = io.Copy(streamWriter, file)
	return err
}

func (s *Server) parseUploadMD(stream pb.StorageService_UploadServer) (path string) {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return
	}
	v := md.Get("path")
	if len(v) > 0 {
		path = v[0]
	}
	return
}
func (s *Server) Upload(stream pb.StorageService_UploadServer) error {
	path := s.parseUploadMD(stream)
	if path == "" {
		return status.Error(codes.InvalidArgument, "missing path")
	}

	exist, err := s.FM.IsFileExist(path)
	if err != nil {
		return err
	}
	if exist {
		return status.Errorf(codes.AlreadyExists, "file %v already exist", path)
	}

	file, err := s.FM.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	streamReader := new(pb.StreamReader)
	streamReader.StorageService_UploadServer(stream)
	_, err = io.Copy(file, streamReader)
	if err != nil {
		return err
	}

	return stream.SendAndClose(new(pb.UploadResponse))
}

// print result of request
func UnaryLogger() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		resp, err = handler(ctx, req)
		if err == nil {
			log.Printf("%v success\n", info.FullMethod)
		} else {
			log.Printf("%v error: %v\n", info.FullMethod, err)
		}
		return resp, err
	}
}
func StreamLogger() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		if err == nil {
			log.Printf("%v success\n", info.FullMethod)
		} else {
			log.Printf("%v error: %v\n", info.FullMethod, err)
		}
		return err
	}
}
