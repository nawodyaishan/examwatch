package store

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// fakeWriter simulates an io.Writer that tracks writes and Sync() calls,
// and errors if buffered beyond one line (simulated by tracking consecutive writes without a sync).
type fakeWriter struct {
	buf        bytes.Buffer
	syncCount  int
	writeCount int
}

func (fw *fakeWriter) Write(p []byte) (n int, err error) {
	fw.writeCount++
	// "errors if buffered beyond one line": if writes outpace syncs by more than 1
	if fw.writeCount-fw.syncCount > 1 {
		return 0, errors.New("buffered beyond one line without Sync")
	}
	return fw.buf.Write(p)
}

func (fw *fakeWriter) Sync() error {
	fw.syncCount++
	return nil
}

func TestJSONLWriter_Sync(t *testing.T) {
	fw := &fakeWriter{}
	jw := NewJSONLWriter(fw)

	// Write first record
	err := jw.Write(map[string]int{"a": 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw.syncCount != 1 {
		t.Fatalf("expected 1 sync, got %d", fw.syncCount)
	}

	// Write second record
	err = jw.Write(map[string]int{"b": 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw.syncCount != 2 {
		t.Fatalf("expected 2 syncs, got %d", fw.syncCount)
	}

	expected := "{\"a\":1}\n{\"b\":2}\n"
	if fw.buf.String() != expected {
		t.Fatalf("expected %q, got %q", expected, fw.buf.String())
	}
}

func TestJSONLReader_PartialTrailingLine(t *testing.T) {
	// Write N lines, kill the writer mid-write (truncate the file)
	data := `{"msg": "line 1"}
{"msg": "line 2"}
{"msg": "partial`

	r := bytes.NewReader([]byte(data))
	jr := NewJSONLReader(r)

	var v map[string]string

	err := jr.Read(&v)
	if err != nil {
		t.Fatalf("unexpected error reading line 1: %v", err)
	}
	if v["msg"] != "line 1" {
		t.Fatalf("expected 'line 1', got %v", v["msg"])
	}

	// Read second valid line
	v = nil
	err = jr.Read(&v)
	if err != nil {
		t.Fatalf("unexpected error reading line 2: %v", err)
	}
	if v["msg"] != "line 2" {
		t.Fatalf("expected 'line 2', got %v", v["msg"])
	}

	// Read third, partial line
	err = jr.Read(&v)
	// We expect EOF because the next line is a partial line
	if err != io.EOF {
		t.Fatalf("expected io.EOF on partial line, got %v", err)
	}
}

func TestJSONLReader_EmptyFile(t *testing.T) {
	r := bytes.NewReader([]byte{})
	jr := NewJSONLReader(r)

	var v map[string]string
	err := jr.Read(&v)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestJSONLReader_EmptyLinesSkipped(t *testing.T) {
	data := `{"msg": "1"}

{"msg": "2"}
`
	r := bytes.NewReader([]byte(data))
	jr := NewJSONLReader(r)

	var v map[string]string

	err := jr.Read(&v)
	if err != nil || v["msg"] != "1" {
		t.Fatalf("expected '1', got err=%v msg=%s", err, v["msg"])
	}

	err = jr.Read(&v)
	if err != nil || v["msg"] != "2" {
		t.Fatalf("expected '2', got err=%v msg=%s", err, v["msg"])
	}

	err = jr.Read(&v)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}
