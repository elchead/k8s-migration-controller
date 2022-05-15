package monitoring_test

import (
	"testing"

	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
)

func TestMaxMigration(t *testing.T) {
	cluster := NewTestCluster()
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"z2_w": 40., "z2_q": 45.}, monitoring.PodMemMap{"z1_w": 10.})
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
	cluster := NewTestCluster()
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"z2_w": 20., "z2_q": 30., "z2_r": 40.}, monitoring.PodMemMap{"z1_w": 10.})
	sut := monitoring.BigEnoughMigrator{cluster, mockClient}
	t.Run("choose smallest pod that is big enough", func(t *testing.T) {
		request := monitoring.NodeFreeGbRequest{Node: "z2", Amount: 26.}
		cmds, err := sut.GetMigrationCmds(request)
		assert.NoError(t, err)
		assert.Equal(t, cmds[0].Usage, 30.)
	})
}

// TODO minimize migration cost (cost/size relationship)
// TODO choose pod with largest slope
