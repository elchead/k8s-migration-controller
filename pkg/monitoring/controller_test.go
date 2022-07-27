package monitoring_test

import (
	"strings"
	"testing"
	"time"

	"github.com/elchead/k8s-cluster-simulator/pkg/clock"
	"github.com/elchead/k8s-migration-controller/pkg/migration"
	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/exp/maps"
)

const testNodeGb = 100.

var testCluster = NewTestCluster()

var ctime = clock.NewClock(time.Now())

func TestControllerRestrictsChoiceForConcurrentChecker(t *testing.T){
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 30., "q_z2": 50.,"z_z2": 20.,}, monitoring.PodMemMap{"w_z1": 10.})
	policy := monitoring.NewThresholdPolicyWithCluster(20., testCluster, mockClient)
	checker := monitoring.NewConcurrentMigrationChecker()
	migrator := monitoring.NewMigrationPolicyWithChecker("optimal",testCluster,mockClient,checker) //&monitoring.OptimalMigrator{Client: mockClient, Cluster: cluster,MinSize:5.,Checker:checker}	
	now := clock.NewClock(time.Now())
	sut := monitoring.NewController(policy,migrator)

	t.Run("starts migration", func(t *testing.T){
		migs,err := sut.GetMigrations(now)
		assert.NoError(t, err)
		assertSingleMigration(t, migs, "z_z2")
		// assert.Equal(t,[]migration.MigrationCmd{{Pod:"z_z2" ,Usage:20 ,NewNode: "z1"}},migs)	
	})
	t.Run("does not repeat same migration", func(t *testing.T){
		migs,err := sut.GetMigrations(now)
		assert.Error(t, err)  // TODO is error desired? check if catched... It's normal.. there is no option, ignore
		assert.Empty(t, migs)
	})
	// t.Run("migrate new pod while still ignoring migrating pod", func(t *testing.T){
	// 	mockClientWithNewPod := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 30., "q_z2": 40.,"z_z2": 20.,"n_z2": 8.}, monitoring.PodMemMap{"w_z1": 10.})
	// 	migratorT := migrator.(monitoring.MigratorAdapter)//.MigrationPolicy.(*monitoring.OptimalMigrator)
	// 	migratorT.ClientRef = monitoring.NewFilteredClient(mockClientWithNewPod) // TODO client ref hacking not working
	// 	migs,err := sut.GetMigrations(now)
	// 	assert.NoError(t, err)
	// 	assertSingleMigration(t, migs, "n_z2")
	// 	// assert.Equal(t,[]migration.MigrationCmd{{Pod:"n_z2" ,Usage:8 ,NewNode:"z1"}},migs) // TODO is error desired? check if catched
	// })
	// t.Run("(hypothetical): allow to migrate old pod when finished migrating pod", func(t *testing.T){
	// 	mockClientWithNewPod := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 30., "q_z2": 40.,"z_z2": 20.,"n_z2": 8.}, monitoring.PodMemMap{"w_z1": 10.})
	// 	migratorT := migrator.(monitoring.MigratorAdapter)//.MigrationPolicy.(*monitoring.OptimalMigrator)
	// 	migratorT.ClientRef = monitoring.NewFilteredClient(mockClientWithNewPod)
	// 	migs,err := sut.GetMigrations(now.Add(1*time.Hour))
	// 	assert.NoError(t, err)
	// 	assertSingleMigration(t, migs, "z_z2")
	// })
}

func assertSingleMigration(t *testing.T, migs []migration.MigrationCmd,migPod string) {
	assert.Len(t, migs,1)
	assert.Equal(t, migs[0].Pod,migPod)
}
func TestControllerBlocksDuringMigration(t *testing.T) {
	mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 30., "q_z2": 50.,"z_z2": 20.}, monitoring.PodMemMap{"w_z1": 10.})
	policy := monitoring.NewThresholdPolicyWithCluster(20., testCluster, mockClient)
	sut := monitoring.NewController(policy,monitoring.NewMigrationPolicy("optimal",testCluster,mockClient)) // TODO test for other mig policies as well
	
	t.Run("starts migration", func(t *testing.T) {
		migs, err := sut.GetMigrations(ctime)
		assert.NoError(t, err)
		assert.NotEmpty(t,migs)
	})

	t.Run("blocks after starting migration", func(t *testing.T) {
		migs, _:= sut.GetMigrations(ctime)	
		assert.Empty(t,migs)
		// assert.Error(t, err)
	})
}

func TestMigration(t *testing.T) {
	t.Run("migrate max pod on critical node", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 42., "q_z2": 45.}, monitoring.PodMemMap{"w_z1": 10.})
		policy := monitoring.NewThresholdPolicyWithCluster(20., testCluster, mockClient)
		sut := monitoring.NewControllerWithPolicy(policy)
		migs, err := sut.GetMigrations(ctime)
		t.Run("migrating node is set in cmd",func(t *testing.T){
			for _,mig := range migs {
				assert.Equal(t,"z1",mig.NewNode)
			}
		})
		assert.NoError(t, err)
		assert.Equal(t, "q_z2", migs[0].Pod)
	})
	t.Run("do not migrate if other node is full", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z2": 30., "q_z2": 30., "z2_z": 30.}, monitoring.PodMemMap{"w_z1": 80.})
		policy := monitoring.NewThresholdPolicyWithCluster(20., testCluster, mockClient)
		sut := monitoring.NewControllerWithPolicy(policy)
		migs, err := sut.GetMigrations(ctime)
		assert.Error(t, err)
		assert.Empty(t, migs)
	})
	t.Run("do not migrate if other node is full after migration", func(t *testing.T) {
		mockClient := setupMockClient(testNodeGb, monitoring.PodMemMap{"w_z1": 27, "q_z2": 30, "z3_t": 30}, monitoring.PodMemMap{"w_z2": 30, "q_z2": 30})
		policy := monitoring.NewThresholdPolicyWithCluster(20., testCluster, mockClient)
		sut := monitoring.NewControllerWithPolicy(policy)
		migs, err := sut.GetMigrations(ctime)
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
	return strings.Split(key, "_")[1]
}

func TestGetNodeNameMock(t *testing.T) {
	assert.Equal(t, "z1", getNodeNameFromPodName(monitoring.PodMemMap{"q_z1": 50.}))
}

func TestGetSumPodMem(t *testing.T) {
	assert.Equal(t, 90., sumPodMemory(monitoring.PodMemMap{"q_z1": 50., "w_z1": 40.}))
}
func TestGetMaxPod(t *testing.T) {
	assert.Equal(t, "q_z1", monitoring.GetMaxPod(monitoring.PodMemMap{"w_z1": 1000, "q_z1": 5000000}))
}

type mockClient struct {
	mock.Mock
}


func (c *mockClient) GetFreeMemoryOfNodes() (monitoring.NodeFreeMemMap, error) {
	args := c.Called()
	return args.Get(0).(monitoring.NodeFreeMemMap), args.Error(1)
}

func (c *mockClient) GetPodMemorySlope(node,podName, time, slopeWindow string) (float64, error) {
	args := c.Called(node,podName,time,slopeWindow)
	return args.Get(0).(float64), args.Error(1)
}

func (c *mockClient) GetFreeMemoryNode(nodeName string) (float64, error) {
	args := c.Called(nodeName)
	return args.Get(0).(float64), args.Error(1)
}

func (c *mockClient) GetPodMemories(nodeName string) (monitoring.PodMemMap, error) {
	args := c.Called(nodeName)
	return args.Get(0).(monitoring.PodMemMap), args.Error(1)
}
