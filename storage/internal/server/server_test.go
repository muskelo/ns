package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/muskelo/ns_server/protos/storage"
	"github.com/muskelo/ns_server/storage/internal/filemanager"
)

var (
	// init by runTestServer
	lis    *bufconn.Listener
	fmPath string
	// init by initTestClient
	client pb.StorageServiceClient
)

func runTestServer() {
	var err error
	fmPath, err = filepath.Abs("../../test/fm")
	if err != nil {
		panic(err)
	}
	lis = bufconn.Listen(1024 * 1024)

	fm := &filemanager.FileManager{Root: fmPath}
	server := New(fm)

	s := grpc.NewServer()
	pb.RegisterStorageServiceServer(s, server)
	if err := s.Serve(lis); err != nil {
		panic(err)
	}
	return
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func initTestClient() {
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	client = pb.NewStorageServiceClient(conn)
}

func cleanup() {
	os.Remove(filepath.Join(fmPath, "dir1/file4.txt"))
    os.RemoveAll(filepath.Join(fmPath, "dir3"))
}

func TestMain(t *testing.M) {
	go runTestServer()
	initTestClient()
	code := t.Run()
	cleanup()
	os.Exit(code)
}

/*
Tests
*/

func TestServer(t *testing.T) {
	err := testReadDir()
	if err != nil {
		t.Errorf("testReadDir() Err: %v", err)
		return
	}

	err = testDownload(t.TempDir())
	if err != nil {
		t.Errorf("testDownload() Err: %v", err)
		return
	}

	err = testUpload()
	if err != nil {
		t.Errorf("testDownload() Err: %v", err)
		return
	}

	err = testRemove()
	if err != nil {
		t.Errorf("testRemove() Err: %v", err)
		return
	}

	err = testMkdir()
	if err != nil {
		t.Errorf("testMkdir() Err: %v", err)
		return
	}

    err = testRemoveAll()
	if err != nil {
		t.Errorf("testMkdir() Err: %v", err)
		return
	}
}

func testReadDir() error {
	wantDirs := []*pb.ReadDirResponse_Dir{
		{
			Name: "dir1",
			Path: "/dir1",
		},
		{
			Name: "dir2",
			Path: "/dir2",
		},
	}
	wantFiles := []*pb.ReadDirResponse_File{
		{
			Name: "file1.txt",
			Path: "/file1.txt",
		},
		{
			Name: "file2.txt",
			Path: "/file2.txt",
		},
	}

	request := &pb.ReadDirRequest{
		Path: "/",
	}
	response, err := client.ReadDir(context.Background(), request)
	if err != nil {
		return err
	}

	for i, dir := range response.Dirs {
		if dir.Name != wantDirs[i].Name || dir.Path != wantDirs[i].Path {
			return fmt.Errorf("dir = %v, want %v", dir, wantDirs[i])
		}
	}
	for i, file := range response.Files {
		if file.Name != wantFiles[i].Name || file.Path != wantFiles[i].Path {
			return fmt.Errorf("file = %v, want %v", file, wantFiles[i])
		}
	}
	return nil
}

func testDownload(tempDir string) error {
	ctx := metadata.AppendToOutgoingContext(context.TODO(), "path", "dir1/file3.txt")
	request := &pb.DownloadRequest{}
	stream, err := client.Download(ctx, request)
	if err != nil {
		return err
	}

	streamReader := new(pb.StreamReader)
	streamReader.StorageService_DownloadClient(stream)

	outPath := filepath.Join(tempDir, "test.txt")
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, streamReader)
	if err != nil {
		return err
	}

	equal, err := compareFiles(filepath.Join(fmPath, "dir1/file3.txt"), outPath)
	if err != nil {
		return err
	}
	if !equal {
		return fmt.Errorf("Downloaded file if different from src")
	}
	return nil
}

func testUpload() error {
	ctx := metadata.AppendToOutgoingContext(context.TODO(), "path", "dir1/file4.txt")
	stream, err := client.Upload(ctx)
	if err != nil {
		return err
	}

	streamWriter := new(pb.StreamWriter)
	streamWriter.StorageService_UploadClient(stream)

	file, err := os.Open("../../test/fm/dir1/file3.txt")
	if err != nil {
		return err
	}

	_, err = io.Copy(streamWriter, file)
	if err != nil {
		return err
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		return err
	}

	ok, err := compareFiles(filepath.Join(fmPath, "/dir1/file3.txt"), filepath.Join(fmPath, "/dir1/file4.txt"))
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Uploaded file if different from src")
	}
	return nil
}

func testRemove() error {
	_, err := client.Remove(context.Background(), &pb.RemoveRequest{Path: "dir1/file4.txt"})
	if err != nil {
		return err
	}

	_, err = os.Stat(filepath.Join(fmPath, "dir1/file4.txt"))
	switch {
	case err == nil:
		return fmt.Errorf("File exist")
	case errors.Is(err, os.ErrNotExist):
		return nil
	default:
		return err
	}
}

func testMkdir() error {
	_, err := client.Mkdir(context.Background(), &pb.MkdirRequest{Path: "/dir3"})
	if err != nil {
		return err
	}
	_, err = os.Stat(filepath.Join(fmPath, "dir3"))
	return err
}

func testRemoveAll() error {
    _, err := client.Mkdir(context.Background(), &pb.MkdirRequest{Path: "/dir3/dir4"})
    if err != nil {
        return err
    }

    _, err = client.RemoveAll(context.Background(), &pb.RemoveAllRequest{Path: "/dir3"})
    if err != nil {
        return err
    }

	_, err = os.Stat(filepath.Join(fmPath, "dir3"))
	switch {
	case err == nil:
		return fmt.Errorf("File exist")
	case errors.Is(err, os.ErrNotExist):
		return nil
	default:
		return err
	}
}

func compareFiles(file1, file2 string) (bool, error) {
	f1, err := os.Open(file1)
	if err != nil {
		return false, err
	}
	f2, err := os.Open(file2)
	if err != nil {
		return false, err
	}

	data1, err := io.ReadAll(f1)
	if err != nil {
		return false, err
	}
	data2, err := io.ReadAll(f2)
	if err != nil {
		return false, err
	}

	return bytes.Equal(data1, data2), nil
}
