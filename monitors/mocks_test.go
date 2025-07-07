package monitors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockPush(t *testing.T) {
	m := NewMockPush()
	assert.Equal(t, 0, m.Calls.Size(), "empty")

	mon := NewMockMonitor("test", "test")
	m.Push(t.Context(), mon, Result{}, Result{})

	assert.Equal(t, 1, m.Calls.Size(), "empty")

	m.ClearCalls()

	assert.Equal(t, 0, m.Calls.Size(), "empty")

}

func TestMockMonitor_Check(t *testing.T) {
	mon := NewMockMonitor("test", "test")
	r := mon.Check(t.Context())
	assert.Equal(t, RAGGreen, r.Status, "green")
	assert.Equal(t, "ok check 1", r.Value, "value")
	r = mon.Check(t.Context())
	assert.Equal(t, RAGGreen, r.Status, "green")
	assert.Equal(t, "ok check 2", r.Value, "value")
	mon.status = []struct {
		rag RAG
		msg string
	}{
		{rag: RAGAmber, msg: "amber"},
		{rag: RAGRed, msg: "red"},
		{rag: RAGGreen, msg: "green"},
		{rag: RAGError, msg: "error"},
	}
	r = mon.Check(t.Context())
	assert.Equal(t, RAGAmber, r.Status, "amber")
	assert.Equal(t, "amber", r.Value, "value")
	assert.Equal(t, 3, len(mon.status), "status len")
}

func TestNewMockMonitor_NameGroup(t *testing.T) {
	mon := NewMockMonitor("test", "group")
	assert.Equal(t, "test", mon.name, "Name")
	assert.Equal(t, "group", mon.group, "Group")
	assert.Equal(t, "MockMonitor test", mon.DisplayName(), "DisplayName")
}
