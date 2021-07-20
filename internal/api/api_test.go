package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteResponse(t *testing.T) {
	w := httptest.NewRecorder()
	response := APIResponse{status: 123, response: "hello world"}
	WriteResponse(w, response)

	result := w.Result()
	body, err := io.ReadAll(result.Body)
	assert.NoError(t, err)
	assert.Equal(t, "application/json", result.Header.Get("Content-Type"))
	assert.Equal(t, 123, w.Code)
	var resultMessage string
	assert.NoError(t, json.Unmarshal(body, &resultMessage))
	assert.Equal(t, "hello world", resultMessage)
}

type unencodable struct{}

func (u *unencodable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("no json for you")
}

func TestWriteResponsePanicsIfNotEncodable(t *testing.T) {
	w := httptest.NewRecorder()
	response := APIResponse{status: 123, response: &unencodable{}}
	assert.Panics(t, func() {
		WriteResponse(w, response)
	})
}
