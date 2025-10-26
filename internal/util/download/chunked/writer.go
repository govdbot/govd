package chunked

import (
	"fmt"
	"io"
)

type ChunkWriter struct {
	writer         io.Writer
	buffer         map[int]*Chunk
	nextIndex      int
	totalChunks    int
	chunksReceived int
}

func newChunkWriter(writer io.Writer, totalChunks int) *ChunkWriter {
	return &ChunkWriter{
		writer:      writer,
		buffer:      make(map[int]*Chunk),
		totalChunks: totalChunks,
	}
}

func (w *ChunkWriter) writeChunk(chunk *Chunk) error {
	defer chunk.reader.Close()

	if _, err := io.Copy(w.writer, chunk.reader); err != nil {
		return fmt.Errorf("failed to write chunk %d: %w", chunk.index, err)
	}
	return nil
}

func (w *ChunkWriter) addChunk(chunk *Chunk) error {
	w.chunksReceived++

	if chunk.err != nil {
		return fmt.Errorf("chunk %d failed: %w", chunk.index, chunk.err)
	}

	w.buffer[chunk.index] = chunk
	return w.flushSequentialChunks()
}

func (w *ChunkWriter) flushSequentialChunks() error {
	for {
		chunk, ready := w.buffer[w.nextIndex]
		if !ready {
			break
		}
		if err := w.writeChunk(chunk); err != nil {
			return err
		}
		delete(w.buffer, w.nextIndex)
		w.nextIndex++
	}
	return nil
}

func (w *ChunkWriter) cleanup() {
	for _, chunk := range w.buffer {
		if chunk.reader != nil {
			chunk.reader.Close()
		}
	}
}

func (w *ChunkWriter) finalize() error {
	if w.chunksReceived != w.totalChunks {
		return fmt.Errorf("expected %d chunks, received %d", w.totalChunks, w.chunksReceived)
	}
	if w.nextIndex != w.totalChunks {
		return fmt.Errorf("expected to write %d chunks, wrote %d", w.totalChunks, w.nextIndex)
	}
	return nil
}
