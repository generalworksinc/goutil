package gw_compress

import (
	"bytes"
	"io"
	"testing"

	"github.com/dsnet/compress/bzip2"
)

func TestCompressAndExtractString(t *testing.T) {
	const src = "hello bzip2"
	buf, err := Compress(bytes.NewReader([]byte(src)))
	if err != nil {
		t.Fatalf("compress error: %v", err)
	}

	// Extract from buffer string form
	out, err := ExtractString(buf.String())
	if err != nil {
		t.Fatalf("extract error: %v", err)
	}
	if out != src {
		t.Fatalf("unexpected extract: %q", out)
	}

	// Reader API
	r, err := Extract(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("reader extract error: %v", err)
	}
	defer r.Close()
	var plain bytes.Buffer
	if _, err := io.Copy(&plain, r); err != nil {
		t.Fatalf("copy error: %v", err)
	}
	if plain.String() != src {
		t.Fatalf("unexpected plain: %q", plain.String())
	}

	// sanity: NewReader on string again
	_, err = bzip2.NewReader(bytes.NewReader(buf.Bytes()), new(bzip2.ReaderConfig))
	if err != nil {
		t.Fatalf("unexpected bzip2 reader error: %v", err)
	}
}
