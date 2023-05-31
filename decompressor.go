package main

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func DecompressAllBody(httpResponse *http.Response) ([]byte, error) {
	contentEncoding, hasContentEncoding := httpResponse.Header["Content-Encoding"]
	var err error
	readCloser := httpResponse.Body
	defer readCloser.Close()
	if hasContentEncoding {
		for _, contentEncodingPart := range contentEncoding {
			encodings := strings.Split(contentEncodingPart, ", ")
			nbEncodings := len(encodings)
			for encodingPos := nbEncodings; encodingPos > 0; encodingPos-- {
				encoding := encodings[encodingPos-1]
				if encoding == "gzip" {
					readCloser, err = gzip.NewReader(readCloser)
					if err != nil {
						return nil, fmt.Errorf("error from gzip.NewReader: %w", err)
					}
					defer readCloser.Close()
				} else if encoding == "compress" {
					return nil, fmt.Errorf("encoding compress is not supported - please use another one")
				} else if encoding == "deflate" {
					readCloser = flate.NewReader(readCloser)
					defer readCloser.Close()
				} else if encoding == "identity" {
					// no compression, do nothing
				} else if encoding == "br" {
					return nil, fmt.Errorf("encoding br is not supported - please use another one")
				} else {
					return nil, fmt.Errorf("unexpected encoding %v", encoding)
				}
			}
		}
	}

	dataBytes, err := io.ReadAll(readCloser)
	if err != nil {
		return nil, fmt.Errorf("error from io.ReadAll: %w", err)
	}
	return dataBytes, nil
}
