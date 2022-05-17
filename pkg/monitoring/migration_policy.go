package monitoring

import (
	"fmt"
	"knapsack/algorithms"

	"github.com/elchead/k8s-migration-controller/pkg/migration"
)

type MigrationPolicy interface {
	GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error)
}


type OptimalMigrator struct {
	Cluster Cluster
	Client  Clienter
}

func (m OptimalMigrator) GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error) {
	podMems, err := m.Client.GetPodMemories(request.Node)
	if err != nil {
		return nil, err
	}
	items := make([]algorithms.Item, 0, len(podMems))

	nameMap := make(map[int]string)
	for name,usage := range podMems {
		nameMap[len(items)] = name
		items = append(items,algorithms.Item{Weight: int(usage),Value: int(usage)})
	}
	capacity := int(request.Amount)
      	_,_,bestConfig := algorithms.KnapsackBruteForce(capacity, items, []int{}, 0, 0, 0)
	
	migrations := make([]migration.MigrationCmd, 0, len(bestConfig))
	for _, idx := range bestConfig {
		pod := nameMap[idx] 
		migrations = append(migrations, migration.MigrationCmd{Pod: pod, Usage: podMems[pod]})
	}
	return migrations, nil
}


type MaxMigrator struct {
	Cluster Cluster
	Client  Clienter
}

func (c MaxMigrator) GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error) {
	podMems, err := c.Client.GetPodMemories(request.Node)
	if err != nil {
		return nil, err
	}
	pod := GetMaxPod(podMems)
	podMem := podMems[pod]
	if podMem < request.Amount {
		err = fmt.Errorf("migration of pod (%f) on node %s does not fullfill request (%f)", podMem,request.Node, request.Amount)
	}
	return []migration.MigrationCmd{{Pod: pod, Usage: podMem}}, err
}

type BigEnoughMigrator struct {
	Cluster Cluster
	Client  Clienter
}

func (c BigEnoughMigrator) GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error) {
	podMems, err := c.Client.GetPodMemories(request.Node)
	if err != nil {
		return nil, err
	}

	pod := GetMinPodBiggerThan(podMems, request.Amount)
	podMem := podMems[pod]
	if podMem < request.Amount {
		err = fmt.Errorf("migration of pod (%f) on node %s does not fullfill request (%f)", podMem,request.Node, request.Amount)
	}
	return []migration.MigrationCmd{{Pod: pod, Usage: podMem}}, err
}

func GetMinPodBiggerThan(pods PodMemMap, amount float64) (pod string) {
	min := 9999.
	for p, mem := range pods {
		if mem < min && mem >= amount {
			min = mem
			pod = p
		}
	}
	return pod
}

func GetMaxPod(pods PodMemMap) (pod string) {
	max := -1.
	for p, mem := range pods {
		if mem > max {
			max = mem
			pod = p
		}
	}
	return pod
}
