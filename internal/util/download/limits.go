package download

import (
	"io"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/util"
)

type sizeLimitedWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *sizeLimitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, util.ErrFileTooLarge
	}
	n, err := w.writer.Write(p)
	w.remaining -= int64(n)
	return n, err
}

func limitedWriter(writer io.Writer) io.Writer {
	return &sizeLimitedWriter{writer: writer, remaining: config.Env.MaxFileSize}
}

func readLimited(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, config.Env.MaxFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > config.Env.MaxFileSize {
		return nil, util.ErrFileTooLarge
	}
	return data, nil
}
