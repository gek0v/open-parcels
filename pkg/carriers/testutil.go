package carriers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// RoundTripFunc allows defining custom http.RoundTripper functions.
type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewFixtureClient returns an http.Client backed by a mock RoundTripper serving fixture content.
func NewFixtureClient(statusCode int, fixtureContent []byte, contentType string) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) *http.Response {
			header := make(http.Header)
			if contentType != "" {
				header.Set("Content-Type", contentType)
			}
			return &http.Response{
				StatusCode: statusCode,
				Status:     http.StatusText(statusCode),
				Body:       io.NopCloser(bytes.NewReader(fixtureContent)),
				Header:     header,
				Request:    req,
			}
		}),
	}
}

// LoadFixture reads a test fixture from testdata/fixtures.
func LoadFixture(filename string) ([]byte, error) {
	candidates := []string{
		filepath.Join("testdata", "fixtures", filename),
		filepath.Join("..", "testdata", "fixtures", filename),
		filepath.Join("..", "..", "testdata", "fixtures", filename),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("could not find fixture %s in candidate paths: %v", filename, candidates)
}
