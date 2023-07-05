package server

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/muskelo/ns_server/protos/storage"
)

func Run(client pb.StorageServiceClient, addr string) error {
	r := gin.Default()
	r.Use(ErrorHandler())
	r.Handle("POST", "/mkdir/", Mkdir(client))
	r.Handle("POST", "/readdir/", ReadDir(client))
	r.Handle("POST", "/remove/", Remove(client))
	r.Handle("POST", "/upload/", Upload(client))
	r.Handle("GET", "/download/", Download(client))
	return r.Run(addr)
}

func Mkdir(client pb.StorageServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		request := &pb.MkdirRequest{}
		err := c.BindJSON(request)
		if err != nil {
			c.Error(&HTTPError{400, "can't parse json"})
			return
		}

		ctx := context.TODO()
		_, err = client.Mkdir(ctx, request)
		if err != nil {
			c.Error(err)
			return
		}
	}
}

type readdirJSON struct {
	Path string `json:"path"`
}

func ReadDir(client pb.StorageServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		request := &pb.ReadDirRequest{}
		err := c.BindJSON(request)
		if err != nil {
			c.Error(&HTTPError{400, "can't parse json"})
			return
		}

		response, err := client.ReadDir(context.Background(), request)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(200, response)
	}
}

type removeJSON struct {
	Path string `json:"path"`
}

func Remove(client pb.StorageServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := removeJSON{}
		err := c.BindJSON(&data)
		if err != nil {
			c.Error(&HTTPError{400, "can't parse json"})
			return
		}

		ctx := context.TODO()
		_, err = client.Remove(ctx, &pb.RemoveRequest{Path: data.Path})
		if err != nil {
			c.Error(err)
			return
		}
	}
}

func setHeadersFromStream(c *gin.Context, stream pb.StorageService_DownloadClient) error {
	md, err := stream.Header()
	if err != nil {
		return err
	}

	v := md.Get("name")
	if len(v) > 0 {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%v", v[0]))
	} else {
		c.Header("Content-Disposition", "attachment; filename=unknow")
	}

	v = md.Get("size")
	if len(v) > 0 {
		c.Header("Accept-Length", v[0])
	}
	return nil
}
func Download(client pb.StorageServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			c.Error(&HTTPError{400, "can't parse json"})
			return
		}

		ctx := metadata.AppendToOutgoingContext(context.TODO(), "path", path)
		stream, err := client.Download(ctx, &pb.DownloadRequest{})
		if err != nil {
			c.Error(err)
			return
		}
		defer stream.CloseSend()

		if err := setHeadersFromStream(c, stream); err != nil {
			c.Error(err)
			return
		}

		r := new(pb.StreamReader)
		r.StorageService_DownloadClient(stream)
		_, err = io.Copy(c.Writer, r)
		if err != nil {
			c.Error(err)
			return
		}
	}
}

func Upload(client pb.StorageServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			c.Error(&HTTPError{400, "path missing"})
			return
		}

		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.Error(&HTTPError{400, "can't parse form"})
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			c.Error(&HTTPError{400, "can't open file"})
			return
		}

		ctx := metadata.AppendToOutgoingContext(context.TODO(), "path", path)
		stream, err := client.Upload(ctx)
		if err != nil {
			c.Error(err)
            return
		}
		defer stream.CloseSend()

		w := new(pb.StreamWriter)
		w.StorageService_UploadClient(stream)
		_, err = io.Copy(w, file)
		if err != nil && !errors.Is(err, io.EOF){
			c.Error(err)
            return
		}

		_, err = stream.CloseAndRecv()
		if err != nil {
            c.Error(err)
		}
	}
}


func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors[0]
		stat, ok := status.FromError(err)
		if ok {
			var httpCode int
			switch stat.Code() {
			case codes.NotFound:
				httpCode = 404
			case codes.AlreadyExists:
				httpCode = 409
			case codes.FailedPrecondition:
				httpCode = 409
			default:
				httpCode = 500
			}
			c.JSON(httpCode, gin.H{
				"msg": stat.Message(),
			})
			return
		}

		switch e := err.Err.(type) {
		case *HTTPError:
			c.JSON(e.Code, gin.H{
				"msg": e.Message,
			})
		default:
			c.JSON(500, gin.H{
				"msg": err.Error(),
			})
		}
	}
}

type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string {
	return e.Message
}
