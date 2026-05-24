package gw_files

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConvFileToByte_DataURI(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte("PNGDATA"))
	reader, ext, err := ConvFileToByte("", "data:image/png;base64,"+data, false, nil)
	if err != nil {
		t.Fatalf("ConvFileToByte error: %v", err)
	}
	if ext != ".png" {
		t.Fatalf("unexpected ext: %s", ext)
	}
	if reader == nil {
		t.Fatalf("nil reader")
	}
}

func TestConvFileToByte_HTTP(t *testing.T) {
	// simple server returning jpeg bytes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(200)
		_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF})
	}))
	defer server.Close()

	reader, ext, err := ConvFileToByte(server.URL+"/x.jpg?query=1", "", false, nil)
	if err != nil {
		t.Fatalf("ConvFileToByte http error: %v", err)
	}
	if ext != ".jpg" {
		t.Fatalf("unexpected ext: %s", ext)
	}
	if reader == nil {
		t.Fatalf("nil reader")
	}
}
