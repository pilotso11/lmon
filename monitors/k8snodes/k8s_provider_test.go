package k8snodes

import (
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

func makeNode(name string, conditions []corev1.NodeCondition, unschedulable bool) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NodeSpec{Unschedulable: unschedulable},
		Status:     corev1.NodeStatus{Conditions: conditions},
	}
}

func TestK8sProvider_GetNodeStatuses_AllReady(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeNode("node-1", []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
		}, false),
		makeNode("node-2", []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
		}, false),
	)

	provider := NewK8sProvider(client)
	statuses, err := provider.GetNodeStatuses(t.Context())
	require.NoError(t, err)
	require.Len(t, statuses, 2)

	for _, s := range statuses {
		assert.True(t, s.Ready)
		assert.False(t, s.Cordoned)
		assert.False(t, s.MemoryPressure)
		assert.False(t, s.DiskPressure)
		assert.False(t, s.PIDPressure)
	}
}

func TestK8sProvider_GetNodeStatuses_NotReady(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeNode("node-1", []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
		}, false),
	)

	provider := NewK8sProvider(client)
	statuses, err := provider.GetNodeStatuses(t.Context())
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	assert.False(t, statuses[0].Ready)
}

func TestK8sProvider_GetNodeStatuses_MemoryPressure(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeNode("node-1", []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
		}, false),
	)

	provider := NewK8sProvider(client)
	statuses, err := provider.GetNodeStatuses(t.Context())
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	assert.True(t, statuses[0].Ready)
	assert.True(t, statuses[0].MemoryPressure)
}

func TestK8sProvider_GetNodeStatuses_DiskPressure(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeNode("node-1", []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue},
		}, false),
	)

	provider := NewK8sProvider(client)
	statuses, err := provider.GetNodeStatuses(t.Context())
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	assert.True(t, statuses[0].DiskPressure)
}

func TestK8sProvider_GetNodeStatuses_PIDPressure(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeNode("node-1", []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			{Type: corev1.NodePIDPressure, Status: corev1.ConditionTrue},
		}, false),
	)

	provider := NewK8sProvider(client)
	statuses, err := provider.GetNodeStatuses(t.Context())
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	assert.True(t, statuses[0].PIDPressure)
}

func TestK8sProvider_GetNodeStatuses_Cordoned(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeNode("node-1", []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
		}, true),
	)

	provider := NewK8sProvider(client)
	statuses, err := provider.GetNodeStatuses(t.Context())
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	assert.True(t, statuses[0].Cordoned)
}

func TestK8sProvider_GetNodeStatuses_MultipleConditions(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeNode("node-1", []corev1.NodeCondition{
			{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
			{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse},
			{Type: corev1.NodePIDPressure, Status: corev1.ConditionFalse},
		}, true),
	)

	provider := NewK8sProvider(client)
	statuses, err := provider.GetNodeStatuses(t.Context())
	require.NoError(t, err)
	require.Len(t, statuses, 1)

	s := statuses[0]
	assert.Equal(t, "node-1", s.Name)
	assert.True(t, s.Ready)
	assert.True(t, s.Cordoned)
	assert.True(t, s.MemoryPressure)
	assert.False(t, s.DiskPressure)
	assert.False(t, s.PIDPressure)
}

func TestK8sProvider_GetNodeStatuses_EmptyCluster(t *testing.T) {
	client := fake.NewSimpleClientset()

	provider := NewK8sProvider(client)
	statuses, err := provider.GetNodeStatuses(t.Context())
	require.NoError(t, err)
	assert.Empty(t, statuses)
}

func TestK8sProvider_GetNodeStatuses_APIError(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("api server down")
	})

	provider := NewK8sProvider(client)
	_, err := provider.GetNodeStatuses(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list nodes:")
	assert.Contains(t, err.Error(), "api server down")
}
