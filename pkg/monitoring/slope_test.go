package monitoring_test

import (
	"testing"

	"github.com/elchead/k8s-migration-controller/pkg/migration"
	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSelectPodWithLar(t *testing.T) {
	cluster := NewTestCluster()
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 40., "q_z2": 45.}, monitoring.PodMemMap{"w_z1": 50., "q_z1": 30.})
	mockClient.On("GetPodMemorySlope", "z2","w_z2",mock.Anything,mock.Anything).Return(3., nil).Once()
	mockClient.On("GetPodMemorySlope", "z2","q_z2",mock.Anything,mock.Anything).Return(1., nil).Once()
	
	sut := monitoring.SlopeMigrator{Cluster:cluster, Client:mockClient,TimeAhead:5.,Buffer:10}
	res,_ := sut.GetMigrationCmds(monitoring.NodeFreeGbRequest{Node:"z2"})
	assertMigration(t,res,"w_z2")


	t.Run("no slope for any pod",func(t *testing.T){
		mockClient.On("GetPodMemorySlope", "z2","w_z2",mock.Anything,mock.Anything).Return(0., nil).Once()
		mockClient.On("GetPodMemorySlope", "z2","q_z2",mock.Anything,mock.Anything).Return(0., nil).Once()
		res,_ := sut.GetMigrationCmds(monitoring.NodeFreeGbRequest{Node:"z2"})
		assert.Empty(t,res)
	})
	t.Run("no migration when buffer not full", func(t *testing.T){
		sut := monitoring.SlopeMigrator{Cluster:cluster, Client:mockClient,TimeAhead:5.,Buffer:10}
		mockClient.On("GetPodMemorySlope", "z2","w_z2",mock.Anything,mock.Anything).Return(1., nil).Once()
		mockClient.On("GetPodMemorySlope", "z2","q_z2",mock.Anything,mock.Anything).Return(.5, nil).Once()	
		res,_ := sut.GetMigrationCmds(monitoring.NodeFreeGbRequest{Node:"z2"})
		assert.Empty(t,res)
		
	})
	t.Run("select all pods for migration so that predicted usage < buffer ",func(t *testing.T){
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 40., "q_z2": 45.,"z_z2":10.}, monitoring.PodMemMap{"w_z1": 50., "q_z1": 30.})		
		sut := monitoring.SlopeMigrator{Cluster:cluster, Client:mockClient,TimeAhead:5.,Buffer:10}
		mockClient.On("GetPodMemorySlope", "z2","w_z2",mock.Anything,mock.Anything).Return(2., nil).Once()
		mockClient.On("GetPodMemorySlope", "z2","q_z2",mock.Anything,mock.Anything).Return(1.5, nil).Once()	
		mockClient.On("GetPodMemorySlope", "z2","z_z2",mock.Anything,mock.Anything).Return(2., nil).Once()
		res,_ := sut.GetMigrationCmds(monitoring.NodeFreeGbRequest{Node:"z2"})
		assertMigration(t,res,"z_z2","w_z2")
	})
}

func assertMigration(t testing.TB,res []migration.MigrationCmd,podName... string) {
	podsLen := len(podName)
	assert.Len(t,res,podsLen)
	for _,pod := range podName {
		inside := false
		for i,_ := range res {
			if res[i].Pod == pod {
				inside = true
			}
		}
		assert.True(t,inside,"migrations do not contain "+pod)
	}
}

