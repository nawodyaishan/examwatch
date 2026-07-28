package store

import (
	"bufio"
	"encoding/json"
	"io"
)

// syncer is an interface that wraps the basic Sync method.
// os.File implements this.
type syncer interface {
	Sync() error
}

// JSONLWriter is an append-only, crash-safe writer for JSONL.
type JSONLWriter struct {
	w   io.Writer
	enc *json.Encoder
}

// NewJSONLWriter creates a new JSONLWriter.
func NewJSONLWriter(w io.Writer) *JSONLWriter {
	return &JSONLWriter{
		w:   w,
		enc: json.NewEncoder(w),
	}
}

// Write writes a single JSON record followed by a newline,
// and immediately flushes/syncs the underlying writer if it supports it.
func (jw *JSONLWriter) Write(v any) error {
	if err := jw.enc.Encode(v); err != nil {
		return err
	}
	if s, ok := jw.w.(syncer); ok {
		return s.Sync()
	}
	return nil
}

// JSONLReader is a reader for JSONL files that skips partial trailing lines
// caused by crashes.
type JSONLReader struct {
	scanner *bufio.Scanner
}

// NewJSONLReader creates a new JSONLReader.
func NewJSONLReader(r io.Reader) *JSONLReader {
	return &JSONLReader{
		scanner: bufio.NewScanner(r),
	}
}

// Read reads the next JSON record into v.
// It returns io.EOF if the end of the file is reached or if a partial trailing line is encountered.
func (jr *JSONLReader) Read(v any) error {
	for jr.scanner.Scan() {
		line := jr.scanner.Bytes()
		// ignore empty lines
		if len(line) == 0 {
			continue
		}

		err := json.Unmarshal(line, v)
		if err != nil {
			// This might be a partial trailing line due to crash.
			// Treat as EOF so the caller processes valid records up to this point.
			return io.EOF
		}
		return nil
	}
	if err := jr.scanner.Err(); err != nil {
		return err
	}
	return io.EOF
}
