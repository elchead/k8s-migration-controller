package monitoring_test

import (
	"testing"
	"time"

	"github.com/elchead/k8s-cluster-simulator/pkg/clock"
	"github.com/elchead/k8s-migration-controller/pkg/migration"
	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
)

var cluster = NewTestCluster()


func TestMigratorBlocksWhenMigInProgress(t *testing.T) {
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"ow_z2": 40., "oq_z2": 45.}, monitoring.PodMemMap{"ow_z1": 10.})
	sut := monitoring.NewMigrationPolicy("optimal",cluster, mockClient)
	now := clock.NewClock(time.Now())
	t.Run("no error before migration start",func(t *testing.T){
		cmds,err :=sut.GetMigrationCmds(now,monitoring.NodeFreeGbRequest{Node: "z2", Amount: 50.})
		assert.NoError(t, err)
		assert.NotEmpty(t,cmds)
	})

	t.Run("throws error after migration start",func(t *testing.T){
		sut.StartMigration(&migration.MigrationCmd{Pod: "ow_z2", Usage: 20.},now)
		cmds,err :=sut.GetMigrationCmds(now,monitoring.NodeFreeGbRequest{Node: "z2", Amount: 50.})
		assert.Error(t, err)
		assert.Empty(t,cmds)
	})
	t.Run("no error after migration",func(t *testing.T){
		sut.StartMigration(&migration.MigrationCmd{Pod: "ow_z2", Usage: 20.},now)
		cmds,err :=sut.GetMigrationCmds(now.Add(1*time.Hour),monitoring.NodeFreeGbRequest{Node: "z2", Amount: 50.})
		assert.NoError(t, err)
		assert.NotEmpty(t,cmds)	
	})
}

func TestConcurrentIgnoresMigratingJobs(t *testing.T){
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"ow_z2": 40., "oq_z2": 45., "oy_z2": 6.}, monitoring.PodMemMap{"ow_z1": 10.})
	now := clock.NewClock(time.Now())
	
	checker := monitoring.NewConcurrentMigrationChecker()
	sut := monitoring.NewMigrationPolicyWithChecker("optimal",cluster, mockClient,checker)

	t.Run("ignores single job",func(t *testing.T) {
		sut.StartMigration(&migration.MigrationCmd{Pod: "oq_z2", Usage: 20.},now)
		cmds,err :=sut.GetMigrationCmds(now,monitoring.NodeFreeGbRequest{Node: "z2", Amount: 50.})
		assert.NoError(t, err)
		assert.ElementsMatch(t, []migration.MigrationCmd{{Pod:"ow_z2",Usage:40.},{Pod:"oy_z2" ,Usage:6} },cmds)
	})
	t.Run("furthermore ignores second job",func(t *testing.T){
		sut.StartMigration(&migration.MigrationCmd{Pod: "ow_z2", Usage: 20.},now)
		cmds,err :=sut.GetMigrationCmds(now,monitoring.NodeFreeGbRequest{Node: "z2", Amount: 50.})
		assert.NoError(t, err)
		assert.Equal(t, []migration.MigrationCmd{{Pod:"oy_z2" ,Usage:6,NewNode:""} },cmds)
	})
}


func TestMaxMigration(t *testing.T) {
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"ow_z2": 40., "oq_z2": 45.}, monitoring.PodMemMap{"ow_z1": 10.})
	sut := monitoring.MaxMigrator{cluster, mockClient}
	t.Run("easy: max pod fulfills request", func(t *testing.T) {
		request := monitoring.NodeFreeGbRequest{Node: "z2", Amount: 41.}
		cmds, err := sut.GetMigrationCmds(request)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, cmds[0].Usage, request.Amount)
	})
	t.Run("max pod does not fulfill request", func(t *testing.T) {
		request := monitoring.NodeFreeGbRequest{Node: "z2", Amount: 60.}
		_, err := sut.GetMigrationCmds(request)
		assert.Error(t, err)
	})
}

func TestBigEnoughMigration(t *testing.T) {
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"ow_z2": 20., "oq_z2": 30., "z2_r": 40.}, monitoring.PodMemMap{"ow_z1": 10.})
	sut := monitoring.NewMigrationPolicy("big-enough",cluster, mockClient)
	t.Run("choose smallest pod that is big enough", func(t *testing.T) {
		request := monitoring.NodeFreeGbRequest{Node: "z2", Amount: 26.}
		cmds, err := sut.GetMigrationCmds(ctime,request)
		assert.NoError(t, err)
		assert.Equal(t, cmds[0].Usage, 30.)
	})
	t.Run("choose multiple smaller pods when no single big pod", func(t *testing.T) {
		request := monitoring.NodeFreeGbRequest{Node: "z2", Amount: 26.}
		cmds, err := sut.GetMigrationCmds(ctime,request)
		assert.NoError(t, err)
		assert.Equal(t, cmds[0].Usage, 30.)
	})
}

func TestKnapsackMigration(t *testing.T) {
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"ow_z2": 20., "oq_z2": 25., "or_z2": 40.}, monitoring.PodMemMap{"ow_z1": 10.})
	sut := monitoring.NewMigrationPolicy("optimal",cluster, mockClient)
	request := monitoring.NodeFreeGbRequest{Node: "z2", Amount: 50.}
	cmds, err := sut.GetMigrationCmds(ctime,request)	
	assert.NoError(t, err)
	assert.Contains(t, cmds, migration.MigrationCmd{Pod: "ow_z2", Usage: 20.})
	assert.Contains(t, cmds, migration.MigrationCmd{Pod: "oq_z2", Usage: 25.})
}

func TestDoNotMigrateSmallJob(t *testing.T) {
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"ow_z2": 20., "oq_z2": 25., "or_z2": 1.,"oy_z2":6.})
	sut := monitoring.NewMigrationPolicy("optimal",cluster, mockClient)
	request := monitoring.NodeFreeGbRequest{Node: "z2", Amount: 50.}
	cmds, err := sut.GetMigrationCmds(ctime,request)	
	assert.NoError(t, err)
	assert.NotContains(t, cmds, migration.MigrationCmd{Pod: "or_z2", Usage: 1.})
}


func TestPunishMigratedJobInOptimalMigrator(t *testing.T) {
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"ow_z2": 20., "oq_z2": 16., "mr_z2": 33.,"ot_z2":17.}, monitoring.PodMemMap{"ow_z1": 10.})
	sut :=  monitoring.NewMigrationPolicy("optimal",cluster, mockClient)
	request := monitoring.NodeFreeGbRequest{Node: "z2", Amount: 50.}
	cmds, err := sut.GetMigrationCmds(ctime,request)	
	assert.NoError(t, err)
	assert.Contains(t, cmds, migration.MigrationCmd{Pod: "ow_z2", Usage: 20.})
 	assert.Contains(t, cmds, migration.MigrationCmd{Pod: "ot_z2", Usage: 17.})
}
