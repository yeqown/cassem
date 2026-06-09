package httpx

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatHeader(t *testing.T) {
	header := http.Header{
		"Accept":        []string{"application/json", "application/grpc"},
		"Authorization": []string{"Bearer token"},
	}

	got := formatHeader(header)

	assert.Contains(t, got, "Accept:application/json;application/grpc ")
	assert.Contains(t, got, "Authorization:Bearer token ")
}
