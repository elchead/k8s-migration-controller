package monitoring_test

import (
	"strings"
	"testing"

	"fmt"

	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/exp/maps"
)

const testNodeGb = 100.

var testCluster = NewTestCluster()

func TestMigration(t *testing.T) {
	t.Run("migrate max pod on critical node", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"z2_w": 40., "z2_q": 45.}, monitoring.PodMemMap{"z1_w": 10.})
		policy := monitoring.NewThresholdPolicyWithCluster(20., testCluster, mockClient)
		sut := monitoring.NewControllerWithPolicy(policy)
		migs, err := sut.GetMigrations()
		assert.NoError(t, err)
		assert.Equal(t, "z2_q", migs[0].Pod)
	})
	t.Run("do not migrate if other node is full", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"z2_w": 30., "z2_q": 30., "z2_z": 30.}, monitoring.PodMemMap{"z1_w": 80.})
		policy := monitoring.NewThresholdPolicyWithCluster(20., testCluster, mockClient)
		sut := monitoring.NewControllerWithPolicy(policy)
		migs, err := sut.GetMigrations()
		assert.Error(t, err)
		assert.Empty(t, migs)
	})
	t.Run("do not migrate if other node is full after migration", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"z1_w": 25, "z2_q": 30, "z3_t": 30}, monitoring.PodMemMap{"z2_w": 30, "z2_q": 30})
		policy := monitoring.NewThresholdPolicyWithCluster(20., testCluster, mockClient)
		sut := monitoring.NewControllerWithPolicy(policy)
		migs, err := sut.GetMigrations()
		assert.Error(t, err)
		assert.Empty(t, migs)
	})
}

func NewTestCluster() monitoring.Cluster {
	return monitoring.Cluster{NbrNodes: 2, NodeGb: testNodeGb}
}

func setupMockClient(nodeGB float64, nodePods ...monitoring.PodMemMap) *mockClient {
	mock := &mockClient{}
	nodeFreeMemMap := monitoring.NodeFreeMemMap{}
	for _, pods := range nodePods {
		node := getNodeNameFromPodName(pods)
		mock.On("GetPodMemories", node).Return(pods, nil)
		nodeFreeMemMap[node] = (nodeGB - sumPodMemory(pods)) / nodeGB * 100.
	}
	fmt.Println(nodeFreeMemMap)
	mock.On("GetFreeMemoryOfNodes").Return(nodeFreeMemMap, nil)
	return mock
}

func sumPodMemory(pods monitoring.PodMemMap) (sum float64) {
	for _, pod := range pods {
		sum += pod
	}
	return
}

func getNodeNameFromPodName(pods monitoring.PodMemMap) string {
	key := maps.Keys(pods)[0]
	return strings.Split(key, "_")[0]
}

func TestGetNodeNameMock(t *testing.T) {
	assert.Equal(t, "z1", getNodeNameFromPodName(monitoring.PodMemMap{"z1_q": 50.}))
}

func TestGetSumPodMem(t *testing.T) {
	assert.Equal(t, 90., sumPodMemory(monitoring.PodMemMap{"z1_q": 50., "z1_w": 40.}))
}
func TestGetMaxPod(t *testing.T) {
	assert.Equal(t, "z1_q", monitoring.GetMaxPod(monitoring.PodMemMap{"z1_w": 1000, "z1_q": 5000000}))
}

type mockClient struct {
	mock.Mock
}

func (c *mockClient) GetFreeMemoryOfNodes() (monitoring.NodeFreeMemMap, error) {
	args := c.Called()
	return args.Get(0).(monitoring.NodeFreeMemMap), args.Error(1)
}

func (c *mockClient) GetFreeMemoryNode(nodeName string) (float64, error) {
	args := c.Called(nodeName)
	return args.Get(0).(float64), args.Error(1)
}

func (c *mockClient) GetPodMemories(nodeName string) (monitoring.PodMemMap, error) {
	args := c.Called(nodeName)
	return args.Get(0).(monitoring.PodMemMap), args.Error(1)
}
