package monitoring_test

import (
	"testing"

	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
)

func TestIsEnoughSpaceAvailable(t *testing.T) {
	cluster := NewTestCluster()
	t.Run("fail if other node would be full after migration", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"z2_w": 40., "z2_q": 45.}, monitoring.PodMemMap{"z1_w": 1., "z1_q": 30.})
		sut := monitoring.NewThresholdPolicyWithCluster(40., cluster, mockClient)
		assert.Equal(t, "", sut.ValidateMigrationsTo("z2", 45.))
	})
	t.Run("succeed if enough space", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"z2_w": 40., "z2_q": 45.}, monitoring.PodMemMap{"z1_w": 1., "z1_q": 2.})
		sut := monitoring.NewThresholdPolicyWithCluster(40., cluster, mockClient)
		assert.Equal(t, "z1", sut.ValidateMigrationsTo("z2", 35.))
	})
}
