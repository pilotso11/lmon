package k8sevents

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestK8sProvider_GetFailureEvents_ReturnsEventsInWindow(t *testing.T) {
	now := time.Now()
	client := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "evt-1", Namespace: "default"},
			Type:           "Warning",
			Reason:         "CrashLoopBackOff",
			Message:        "Back-off restarting",
			InvolvedObject: corev1.ObjectReference{Name: "pod-1"},
			LastTimestamp:   metav1.NewTime(now.Add(-30 * time.Second)),
		},
	)

	provider := NewK8sProvider(client)
	events, err := provider.GetFailureEvents(t.Context(), "default", 300)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "pod-1", events[0].Pod)
	assert.Equal(t, "CrashLoopBackOff", events[0].Reason)
	assert.Equal(t, "Back-off restarting", events[0].Message)
	assert.Equal(t, "default", events[0].Namespace)
}

func TestK8sProvider_GetFailureEvents_FiltersNonFailureReasons(t *testing.T) {
	now := time.Now()
	client := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "evt-scheduled", Namespace: "default"},
			Type:           "Warning",
			Reason:         "Scheduled",
			Message:        "Successfully assigned",
			InvolvedObject: corev1.ObjectReference{Name: "pod-1"},
			LastTimestamp:   metav1.NewTime(now.Add(-10 * time.Second)),
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "evt-pulling", Namespace: "default"},
			Type:           "Warning",
			Reason:         "Pulling",
			Message:        "Pulling image",
			InvolvedObject: corev1.ObjectReference{Name: "pod-2"},
			LastTimestamp:   metav1.NewTime(now.Add(-10 * time.Second)),
		},
	)

	provider := NewK8sProvider(client)
	events, err := provider.GetFailureEvents(t.Context(), "default", 300)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestK8sProvider_GetFailureEvents_FiltersOutsideWindow(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "old-event", Namespace: "default"},
			Type:           "Warning",
			Reason:         "CrashLoopBackOff",
			Message:        "Old crash",
			InvolvedObject: corev1.ObjectReference{Name: "pod-1"},
			LastTimestamp:   metav1.NewTime(time.Now().Add(-1 * time.Hour)),
		},
	)

	provider := NewK8sProvider(client)
	events, err := provider.GetFailureEvents(t.Context(), "default", 300) // 5 min window
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestK8sProvider_GetFailureEvents_EventTimeFallback(t *testing.T) {
	now := time.Now()
	event := &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: "evt-eventtime", Namespace: "default"},
		Type:           "Warning",
		Reason:         "OOMKilled",
		Message:        "OOM",
		InvolvedObject: corev1.ObjectReference{Name: "pod-1"},
		// LastTimestamp is zero, EventTime is used
		EventTime: metav1.NewMicroTime(now.Add(-10 * time.Second)),
	}
	client := fake.NewSimpleClientset(event)

	provider := NewK8sProvider(client)
	events, err := provider.GetFailureEvents(t.Context(), "default", 300)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "OOMKilled", events[0].Reason)
}

func TestK8sProvider_GetFailureEvents_AllFailureReasons(t *testing.T) {
	now := time.Now()
	reasons := []string{
		"CrashLoopBackOff", "OOMKilled", "Failed", "BackOff",
		"Unhealthy", "FailedMount", "FailedScheduling", "Evicted",
	}

	var objects []runtime.Object
	for i, reason := range reasons {
		objects = append(objects, &corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: fmt.Sprintf("evt-%d", i), Namespace: "default"},
			Type:           "Warning",
			Reason:         reason,
			Message:        fmt.Sprintf("msg-%s", reason),
			InvolvedObject: corev1.ObjectReference{Name: fmt.Sprintf("pod-%d", i)},
			LastTimestamp:   metav1.NewTime(now.Add(-10 * time.Second)),
		})
	}

	client := fake.NewSimpleClientset(objects...)
	provider := NewK8sProvider(client)
	events, err := provider.GetFailureEvents(t.Context(), "default", 300)
	require.NoError(t, err)
	assert.Len(t, events, 8)

	returnedReasons := make(map[string]bool)
	for _, e := range events {
		returnedReasons[e.Reason] = true
	}
	for _, reason := range reasons {
		assert.True(t, returnedReasons[reason], "expected reason %s to be returned", reason)
	}
}

func TestK8sProvider_GetFailureEvents_EmptyEvents(t *testing.T) {
	client := fake.NewSimpleClientset()

	provider := NewK8sProvider(client)
	events, err := provider.GetFailureEvents(t.Context(), "default", 300)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestK8sProvider_GetFailureEvents_APIError(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "events", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("api server unavailable")
	})

	provider := NewK8sProvider(client)
	_, err := provider.GetFailureEvents(t.Context(), "default", 300)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list events:")
	assert.Contains(t, err.Error(), "api server unavailable")
}
