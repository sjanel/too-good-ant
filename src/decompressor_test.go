package tga

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"testing"
)

const kSomeData = "This is a string of data which will be compressed and uncompressed during this unit test"

func TestHttpDecompressorGzip(t *testing.T) {

	body, err := gzipCompress([]byte(kSomeData))
	bodyCloser := io.NopCloser(bytes.NewReader(body))

	httpResponse := &http.Response{
		Header: map[string][]string{
			"Content-Encoding": {
				"identity, identity, gzip, identity",
			},
		},
		Body: bodyCloser,
	}
	data, err := DecompressAllBody(httpResponse)
	if err != nil {
		t.Fatalf("error from DecompressAllBody: %v", err)
	}
	if string(data) != kSomeData {
		t.Fatalf("expected '%v', got %v", kSomeData, string(data))
	}
}

func TestHttpDecompressorGzipFlate(t *testing.T) {

	body, err := gzipCompress([]byte(kSomeData))
	body, err = flateCompress(body)
	bodyCloser := io.NopCloser(bytes.NewReader(body))

	httpResponse := &http.Response{
		Header: map[string][]string{
			"Content-Encoding": {
				"gzip, deflate",
			},
		},
		Body: bodyCloser,
	}
	data, err := DecompressAllBody(httpResponse)
	if err != nil {
		t.Fatalf("error from DecompressAllBody: %v", err)
	}
	if string(data) != kSomeData {
		t.Fatalf("expected '%v', got %v", kSomeData, string(data))
	}
}

func gzipCompress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("error from gz.Write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("error from gz.Close: %w", err)
	}

	return b.Bytes(), nil
}

func gzipDecompress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(data)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return nil, fmt.Errorf("error from gzip.NewReader: %w", err)
	}
	bytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error from io.ReadAll: %w", err)
	}
	return bytes, nil
}

func flateCompress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := flate.NewWriter(&b, -1)
	if err != nil {
		return nil, fmt.Errorf("error from flate.NewWriter: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("error from w.Write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("error from w.Close: %w", err)
	}

	return b.Bytes(), nil
}

func flateDecompress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(data)
	r := flate.NewReader(buf)
	bytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error from io.ReadAll: %w", err)
	}
	return bytes, nil
}
