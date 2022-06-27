package monitoring_test

import (
	"testing"

	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSlopeRequester(t *testing.T) {
	cluster := NewTestCluster()
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 40., "q_z2": 45.}, monitoring.PodMemMap{"w_z1": 50., "q_z1": 30.})

	t.Run("request 7.5gb on z2 when 10gb should be free, 15 gb are free and predicted usage is 12.5gb. request on z1 too ", func(t *testing.T) {
		mockClient.On("GetPodMemorySlope", "z1","w_z1",mock.Anything,mock.Anything).Return(3., nil)
		mockClient.On("GetPodMemorySlope", "z1","q_z1",mock.Anything,mock.Anything).Return(0., nil)
		mockClient.On("GetPodMemorySlope", "z2","w_z2",mock.Anything,mock.Anything).Return(1.5, nil)
		mockClient.On("GetPodMemorySlope", "z2","q_z2",mock.Anything,mock.Anything).Return(1., nil)
		
		sut := monitoring.NewSlopePolicyWithClusterAndTime(10.,5.,cluster, mockClient)
		assert.ElementsMatch(t,[]monitoring.NodeFreeGbRequest([]monitoring.NodeFreeGbRequest{{Node:"z2", Amount:7.5},{Node:"z1",Amount:5.}}),sut.GetNodeFreeGbRequests())
	})
}

func TestSelectNodeWithHighestAvailableMemory(t *testing.T) {
	cluster := NewTestCluster()
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 40., "q_z2": 45.}, monitoring.PodMemMap{"w_z1": 50., "q_z1": 30.},monitoring.PodMemMap{"w_z3": 10., "q_z3": 5.})

	t.Run("migrate to node with least usage", func(t *testing.T) {
		sut := monitoring.NewSlopePolicyWithClusterAndTime(10.,5.,cluster, mockClient)
		migratingNode := sut.ValidateMigrationsTo("z2",7.5)	
		assert.Equal(t,"z3",migratingNode)

	})	
}

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
