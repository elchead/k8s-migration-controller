package monitoring_test

import (
	"testing"

	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
)

func TestIsEnoughSpaceAvailable(t *testing.T) {
	cluster := NewTestCluster()
	t.Run("fail if other node would be full after migration", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 40., "q_z2": 45.}, monitoring.PodMemMap{"w_z1": 1., "q_z1": 30.})
		sut := monitoring.NewThresholdPolicyWithCluster(40., cluster, mockClient)
		assert.Equal(t, "", sut.ValidateMigrationsTo("z2", 45.))
	})
	t.Run("succeed if enough space", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 40., "q_z2": 45.}, monitoring.PodMemMap{"w_z1": 1., "q_z1": 2.})
		sut := monitoring.NewThresholdPolicyWithCluster(40., cluster, mockClient)
		assert.Equal(t, "z1", sut.ValidateMigrationsTo("z2", 35.))
	})
}

func TestGetNodeFreeRequest(t *testing.T) {
	cluster := NewTestCluster()
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 40., "q_z2": 45.}, monitoring.PodMemMap{"w_z1": 1., "q_z1": 30.})
	sut := monitoring.NewThresholdPolicyWithCluster(40., cluster, mockClient)
	assert.Equal(t,[]monitoring.NodeFreeGbRequest([]monitoring.NodeFreeGbRequest{{Node:"z2", Amount:25}}),sut.GetNodeFreeGbRequests())	
}
