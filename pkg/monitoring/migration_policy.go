package monitoring

import (
	"fmt"
	"knapsack/algorithms"
	"log"
	"math"

	"github.com/elchead/k8s-migration-controller/pkg/migration"
)


func NewMigrationPolicy(policy string, cluster Cluster,client Clienter) MigrationPolicy {
	switch policy {
	case "optimal":
		return &OptimalMigrator{Client: client, Cluster: cluster}
	case "max":
		return &MaxMigrator{Cluster: cluster, Client: client}
	case "big-enough":
		return &BigEnoughMigrator{Cluster: cluster, Client: client}
	default:
		log.Println("Defaulting to optimal migration policy. Unknown policy: ",policy)
		return &OptimalMigrator{Cluster: cluster, Client: client}
	}
}

type MigrationPolicy interface {
	GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error)
}


type OptimalMigrator struct {
	Cluster Cluster
	Client  Clienter
}

func removeDuplicateInt(intSlice []int) []int {
	allKeys := make(map[int]bool)
	list := []int{}
	for _, item := range intSlice {
	    if _, value := allKeys[item]; !value {
		allKeys[item] = true
		list = append(list, item)
	    }
	}
	return list
    }
    

func (m OptimalMigrator) GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error) {
	podMems, err := m.Client.GetPodMemories(request.Node)
	if err != nil {
		return nil, err
	}
	items, nameMap := createItemsAndNameMap(podMems)
	capacity := int(request.Amount)
      	_,_,bestConfig := algorithms.KnapsackBruteForce(capacity, items, []int{}, 0, 0, 0.)
	bestConfig = removeDuplicateInt(bestConfig)
	// log.Print("migrator optimal config: ",bestConfig)
	// _,bestConfig := algorithms.KnapsackDynamicWeight(capacity, items,)
	
	migrations := make([]migration.MigrationCmd, 0, len(bestConfig))
	for _, idx := range bestConfig {
		pod := nameMap[idx] 
		migrations = append(migrations, migration.MigrationCmd{Pod: pod, Usage: podMems[pod]})
	}
	return migrations, nil
}

func createItemsAndNameMap(podMems PodMemMap) ([]algorithms.FItem, map[int]string) {
	items := make([]algorithms.FItem, 0, len(podMems))

	nameMap := make(map[int]string)
	for name, usage := range podMems {
		if usage > 0 {
			nameMap[len(items)] = name
			nbrMigrations,err := podMems.CountMigrations(name)
			if err != nil {
				log.Printf("Could not find pod %s to evaluate it's migration cost. Skipping it in migration decision", name)
				continue
			}
			value := getValueWithPunishedMigration(nbrMigrations,usage)
			items = append(items, algorithms.FItem{Weight: int(usage), Value: value})
		}
	}
	return items, nameMap
}

func getValueWithPunishedMigration(nbrMigrations int, size float64) float64 {
	return math.Pow(.5,float64(nbrMigrations))*size
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
