package storage

// adapter stream to io.Write interface
type StreamWriter struct {
	callback func([]byte) (int, error)
}

func (w *StreamWriter) Write(buf []byte) (int, error) {
	return w.callback(buf)
}

func (w *StreamWriter) StorageService_DownloadServer(stream StorageService_DownloadServer) {
	w.callback = func(b []byte) (int, error) {
		response := &DownloadResponse{
			Chunk: b,
		}
		err := stream.Send(response)
		return len(b), err
	}
}
func (w *StreamWriter) StorageService_UploadClient(stream StorageService_UploadClient) {
	w.callback = func(b []byte) (int, error) {
		request := &UploadRequest{
			Chunk: b,
		}
		err := stream.Send(request)
		return len(b), err
	}
}

// adapter stream to io.Reader interface
type StreamReader struct {
	callback func([]byte) (int, error)
}

func (r *StreamReader) Read(b []byte) (int, error) {
	return r.callback(b)
}

func (r *StreamReader) StorageService_DownloadClient(stream StorageService_DownloadClient) {
	r.callback = func(b []byte) (int, error) {
		response, err := stream.Recv()
		if err != nil {
			return 0, err
		}
		for i := 0; i < len(response.Chunk); i++ {
			b[i] = response.Chunk[i]
		}
		return len(response.Chunk), nil
	}
}

func (r *StreamReader) StorageService_UploadServer(stream StorageService_UploadServer) {
	r.callback = func(b []byte) (int, error) {
		request, err := stream.Recv()
		if err != nil {
			return 0, err
		}
		for i := 0; i < len(request.Chunk); i++ {
			b[i] = request.Chunk[i]
		}
		return len(request.Chunk), nil
	}
}
