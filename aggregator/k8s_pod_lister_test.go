package aggregator

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestK8sPodLister_RunningPodsReturned(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.1"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-2", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.2"},
		},
	)

	lister := NewK8sPodLister(client, "default")
	endpoints, err := lister.ListPods(t.Context(), "app=lmon")
	require.NoError(t, err)
	assert.Len(t, endpoints, 2)
	assert.Equal(t, "pod-1", endpoints[0].Name)
	assert.Equal(t, "10.0.0.1", endpoints[0].Address)
	assert.Equal(t, "pod-2", endpoints[1].Name)
	assert.Equal(t, "10.0.0.2", endpoints[1].Address)
}

func TestK8sPodLister_PendingPodsFiltered(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "running", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.1"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pending", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodPending, PodIP: "10.0.0.2"},
		},
	)

	lister := NewK8sPodLister(client, "default")
	endpoints, err := lister.ListPods(t.Context(), "app=lmon")
	require.NoError(t, err)
	assert.Len(t, endpoints, 1)
	assert.Equal(t, "running", endpoints[0].Name)
}

func TestK8sPodLister_PodsWithoutIPFiltered(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "with-ip", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.1"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "no-ip", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: ""},
		},
	)

	lister := NewK8sPodLister(client, "default")
	endpoints, err := lister.ListPods(t.Context(), "app=lmon")
	require.NoError(t, err)
	assert.Len(t, endpoints, 1)
	assert.Equal(t, "with-ip", endpoints[0].Name)
}

func TestK8sPodLister_MixedPodStates(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "good-1", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.1"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pending", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodPending, PodIP: "10.0.0.2"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "no-ip", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "good-2", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.3"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "failed", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodFailed, PodIP: "10.0.0.4"},
		},
	)

	lister := NewK8sPodLister(client, "default")
	endpoints, err := lister.ListPods(t.Context(), "app=lmon")
	require.NoError(t, err)
	assert.Len(t, endpoints, 2)
	assert.Equal(t, "good-1", endpoints[0].Name)
	assert.Equal(t, "good-2", endpoints[1].Name)
}

func TestK8sPodLister_EmptyResult(t *testing.T) {
	client := fake.NewSimpleClientset()

	lister := NewK8sPodLister(client, "default")
	endpoints, err := lister.ListPods(t.Context(), "app=lmon")
	require.NoError(t, err)
	assert.Empty(t, endpoints)
	assert.NotNil(t, endpoints)
}

func TestK8sPodLister_APIErrorPropagated(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("api unavailable")
	})

	lister := NewK8sPodLister(client, "default")
	_, err := lister.ListPods(t.Context(), "app=lmon")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list pods:")
	assert.Contains(t, err.Error(), "api unavailable")
}

func TestK8sPodLister_NamespaceScoping(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-default", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.1"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-system", Namespace: "kube-system", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.2"},
		},
	)

	lister := NewK8sPodLister(client, "kube-system")
	endpoints, err := lister.ListPods(t.Context(), "app=lmon")
	require.NoError(t, err)
	assert.Len(t, endpoints, 1)
	assert.Equal(t, "pod-system", endpoints[0].Name)
}

func TestK8sPodLister_AllNamespaces(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-default", Namespace: "default", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.1"},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-system", Namespace: "kube-system", Labels: map[string]string{"app": "lmon"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.2"},
		},
	)

	// Empty namespace means all namespaces
	lister := NewK8sPodLister(client, "")
	endpoints, err := lister.ListPods(context.Background(), "app=lmon")
	require.NoError(t, err)
	assert.Len(t, endpoints, 2)
}
