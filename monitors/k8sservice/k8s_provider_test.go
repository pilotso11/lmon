package k8sservice

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// testServerPort returns the port of an httptest server.
func testServerPort(t *testing.T, srv *httptest.Server) int {
	t.Helper()
	return srv.Listener.Addr().(*net.TCPAddr).Port
}

func TestK8sProvider_GetPodHealth_AllHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := testServerPort(t, srv)

	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "default"},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{IP: "127.0.0.1", TargetRef: &corev1.ObjectReference{Name: "pod-1"}},
					{IP: "127.0.0.1", TargetRef: &corev1.ObjectReference{Name: "pod-2"}},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(endpoints)
	provider := NewK8sProvider(client)

	pods, err := provider.GetPodHealth(t.Context(), "default", "my-svc", port, "/healthz", 5000)
	require.NoError(t, err)
	require.Len(t, pods, 2)
	assert.True(t, pods[0].Healthy)
	assert.True(t, pods[1].Healthy)
	assert.Equal(t, "pod-1", pods[0].Name)
	assert.Equal(t, "pod-2", pods[1].Name)
}

func TestK8sProvider_GetPodHealth_UnhealthyPod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	port := testServerPort(t, srv)

	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "default"},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{IP: "127.0.0.1", TargetRef: &corev1.ObjectReference{Name: "pod-1"}},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(endpoints)
	provider := NewK8sProvider(client)

	pods, err := provider.GetPodHealth(t.Context(), "default", "my-svc", port, "/healthz", 5000)
	require.NoError(t, err)
	require.Len(t, pods, 1)
	assert.False(t, pods[0].Healthy)
}

func TestK8sProvider_GetPodHealth_NoTargetRef(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := testServerPort(t, srv)

	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "default"},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{IP: "127.0.0.1"}, // no TargetRef
				},
			},
		},
	}

	client := fake.NewSimpleClientset(endpoints)
	provider := NewK8sProvider(client)

	pods, err := provider.GetPodHealth(t.Context(), "default", "my-svc", port, "/healthz", 5000)
	require.NoError(t, err)
	require.Len(t, pods, 1)
	assert.Equal(t, "", pods[0].Name)
	assert.True(t, pods[0].Healthy)
}

func TestK8sProvider_GetPodHealth_EndpointNotFound(t *testing.T) {
	client := fake.NewSimpleClientset() // no endpoints

	provider := NewK8sProvider(client)
	_, err := provider.GetPodHealth(t.Context(), "default", "nonexistent", 8080, "/healthz", 5000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get endpoints for")
}

func TestK8sProvider_GetPodHealth_EmptySubsets(t *testing.T) {
	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "default"},
		Subsets:    []corev1.EndpointSubset{},
	}

	client := fake.NewSimpleClientset(endpoints)
	provider := NewK8sProvider(client)

	pods, err := provider.GetPodHealth(t.Context(), "default", "my-svc", 8080, "/healthz", 5000)
	require.NoError(t, err)
	assert.Empty(t, pods)
}

func TestK8sProvider_GetPodHealth_Non2xxUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer srv.Close()

	port := testServerPort(t, srv)

	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "default"},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{IP: "127.0.0.1", TargetRef: &corev1.ObjectReference{Name: "pod-1"}},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(endpoints)
	provider := NewK8sProvider(client)
	// Disable redirect following so 301 is treated as the final response
	provider.httpClient = &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	pods, err := provider.GetPodHealth(t.Context(), "default", "my-svc", port, "/healthz", 5000)
	require.NoError(t, err)
	require.Len(t, pods, 1)
	assert.False(t, pods[0].Healthy)
}

func TestK8sProvider_GetPodHealth_APIError(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("get", "endpoints", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("api unavailable")
	})

	provider := NewK8sProvider(client)
	_, err := provider.GetPodHealth(t.Context(), "default", "my-svc", 8080, "/healthz", 5000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get endpoints for")
	assert.Contains(t, err.Error(), "api unavailable")
}
