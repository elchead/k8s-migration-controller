package monitoring_test

import (
	"testing"

	"github.com/elchead/k8s-migration-controller/pkg/migration"
	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSingleMigrationS(t *testing.T) {
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"ow_z2": 20., "oq_z2": 35.}, monitoring.PodMemMap{"ow_z1": 10.})


	t.Run("two migs without SingleMigration", func(t *testing.T) {
		sut := monitoring.NewThresholdPolicyWithCluster(30., cluster, mockClient)
		res := sut.ValidateCmds("z2",[]migration.MigrationCmd{{Pod:"ow_z2",Usage:20.},{Pod:"oq_z2",Usage:35}})
		assert.Len(t,res,2)
	})
	t.Run("single mig with SingleMigration",func(t *testing.T) {
		sut := monitoring.NewSingleThresholdPolicyWithCluster(30., cluster, mockClient)
		res := sut.ValidateCmds("z2",[]migration.MigrationCmd{{Pod:"ow_z2",Usage:20.},{Pod:"oq_z2",Usage:35}})
		assert.Len(t,res,1)	
	})
}

func TestThresholdRequesterValidatesAsMuchAsPossible(t *testing.T){
	cluster := NewTestCluster()
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 60., "q_z2": 40.}, monitoring.PodMemMap{"w_z1": 1., "q_z1": 30.},monitoring.PodMemMap{"w_z3": 30., "q_z3": 5.})
	sut := monitoring.NewThresholdPolicyWithCluster(10., cluster, mockClient)
	cmds := []migration.MigrationCmd{{Pod:"w_z2",Usage:60.},{Pod:"q_z2",Usage:40}} // are they always sorted?? if not enough space skip, and try next
	newcmds := sut.ValidateCmds("z2",cmds)
	t.Run("choose as much as possible on highest available node", func(t *testing.T) {
		assert.Equal(t,[]migration.MigrationCmd{{Pod:"q_z2",Usage:40,NewNode:"z1"}},newcmds)
	})
	
	t.Run("same behavior for slope requester",func(t *testing.T){
		cluster := NewTestCluster()
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 60., "q_z2": 40.}, monitoring.PodMemMap{"w_z1": 1., "q_z1": 30.})
		sut := monitoring.NewSlopePolicyWithCluster(10., cluster, mockClient)
		cmds := []migration.MigrationCmd{{Pod:"w_z2",Usage:60.},{Pod:"q_z2",Usage:40}} // are they always sorted?? if not enough space skip, and try next
		newcmds := sut.ValidateCmds("z2",cmds)
		assert.Equal(t,[]migration.MigrationCmd{{Pod:"q_z2",Usage:40,NewNode:"z1"}},newcmds)		
	})
}

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

func TestGetNodeFreeRequest(t *testing.T) {
	cluster := NewTestCluster()
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 40., "q_z2": 45.}, monitoring.PodMemMap{"w_z1": 1., "q_z1": 30.})
	sut := monitoring.NewThresholdPolicyWithCluster(40., cluster, mockClient)
	assert.Equal(t,[]monitoring.NodeFreeGbRequest([]monitoring.NodeFreeGbRequest{{Node:"z2", Amount:25}}),sut.GetNodeFreeGbRequests())	
}
