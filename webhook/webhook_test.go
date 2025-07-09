package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"lmon/common"
)

func TestSend(t *testing.T) {
	ts := common.StartTestServer(t, "/webhook")

	err := Send(t.Context(), ts.URL, `test message`)
	assert.NoError(t, err, "should not error")

	assert.Equal(t, `{"text":"test message"}`, ts.ReqBody.Load(), "should send correct message")
	assert.Equal(t, "application/json", ts.BodyType.Load(), "should set correct content type")
}

func TestSendToBad(t *testing.T) {
	ts := common.StartTestServer(t, "/webhook")

	err := Send(t.Context(), ts.URL+"/hook2", `test message`)
	assert.Error(t, err, "should not error")
}

func TestSendToMissing(t *testing.T) {
	err := Send(t.Context(), "http://localhost:55332", `test message`)
	assert.Error(t, err, "should not error")
}
