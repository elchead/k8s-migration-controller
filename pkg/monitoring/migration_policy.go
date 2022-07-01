package monitoring

import (
	"errors"
	"fmt"
	"knapsack/algorithms"
	"log"
	"math"
	"time"

	"github.com/elchead/k8s-cluster-simulator/pkg/clock"
	"github.com/elchead/k8s-migration-controller/pkg/migration"
)


func NewMigrationPolicy(policy string, cluster Cluster,client Clienter) MigrationPolicy {
	checker := NewBlockingMigrationChecker()
	switch policy {
	case "optimal":
		return &OptimalMigrator{Client: client, Cluster: cluster,MinSize:5.,Checker:checker}
	case "max":
		return &MaxMigrator{Cluster: cluster, Client: client}
	case "big-enough":
		return &BigEnoughMigrator{Cluster: cluster, Client: client}
	default:
		log.Println("Defaulting to optimal migration policy. Unknown policy: ",policy)
		return &OptimalMigrator{Cluster: cluster, Client: client,MinSize:5.}
	}
}

type MigrationPolicy interface {
	GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error)
	// GetMigrationCmdsWithTime(now clock.Clock, request NodeFreeGbRequest) ([]migration.MigrationCmd, error)
	StartMigration(migration.MigrationCmd,clock.Clock)
}

type MigrationPolicyNew interface {
	// GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error)
	GetMigrationCmdsWithTime(now clock.Clock, request NodeFreeGbRequest) ([]migration.MigrationCmd, error)
	StartMigration(migration.MigrationCmd,clock.Clock)
}

type MigratorAdapter struct {
	MigrationPolicy
	Checker MigrationCheckerI
}

// type

func (m MigratorAdapter) GetMigrationCmdsWithTime(now clock.Clock,request NodeFreeGbRequest) ([]migration.MigrationCmd, error) {
	if !m.Checker.IsReady(now) {
		return nil, errors.New("checker not ready")
	}
	// jobsToIgnore := m.Checker.GetMigratingJobs(now) // TODO
	return m.MigrationPolicy.GetMigrationCmds(request)
// filtered := filterPods(podMems, jobsToIgnore)
}

func filterPods(podMems PodMemMap, jobsToIgnore []string) ( PodMemMap) {
	filteredPodMems := PodMemMap{}
	for job, amount := range podMems {
		if !contains(jobsToIgnore, job) {
			filteredPodMems[job] = amount
		}
	}
	return filteredPodMems
}

func (m MigratorAdapter) StartMigration(cmd migration.MigrationCmd,now clock.Clock) {
	m.Checker.StartMigration(now,cmd.Usage,cmd.Pod)
}

func (m OptimalMigrator) StartMigration(cmd migration.MigrationCmd,now clock.Clock) {
	// now := clock.NewClock(time.Now()) // TODO 
	m.Checker.StartMigration(now,cmd.Usage,cmd.Pod)
}

type OptimalMigrator struct {
	Cluster Cluster
	Client  Clienter
	MinSize float64
	Checker MigrationCheckerI
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
	now := clock.NewClock(time.Now()) // TODO use pseudo time in controller (sync with sim time?)
	if !m.Checker.IsReady(now) {
		return nil, errors.New("checker not ready")
	}
	podMems, err := m.Client.GetPodMemories(request.Node)
	if err != nil {
		return nil, err
	}
	m.Checker.GetMigratingJobs(now)
	items, nameMap := createItemsAndNameMap(podMems,m.MinSize)

	// fmt.Printf("ITEMS: %+v\n",items)
	capacity := int(request.Amount)
      	_,_,bestConfig := algorithms.KnapsackBruteForce(capacity, items, []int{}, 0, 0, 0.)
	bestConfig = removeDuplicateInt(bestConfig)
	if len(bestConfig) == 0 {
		return nil, errors.New("no migration pod found. Is smallest pod bigger than requested amount?")
	}
	// log.Print("migrator optimal config: ",bestConfig)
	// fmt.Println("MAP",nameMap)
	// _,bestConfig := algorithms.KnapsackDynamicWeight(capacity, items,)
	
	migrations := make([]migration.MigrationCmd, 0, len(bestConfig))
	for _, idx := range bestConfig {
		pod := nameMap[idx] 
		migrations = append(migrations, migration.MigrationCmd{Pod: pod, Usage: podMems[pod]})
	}
	return migrations, nil
}

func (m OptimalMigrator) GetMigrationCmdsWithTime(now clock.Clock,request NodeFreeGbRequest) ([]migration.MigrationCmd, error) {
	if !m.Checker.IsReady(now) {
		return nil, errors.New("checker not ready")
	}
	podMems, err := m.Client.GetPodMemories(request.Node)
	if err != nil {
		return nil, err
	}
	ignoreJobs := m.Checker.GetMigratingJobs(now)
	filteredPodMems := filterPods(podMems, ignoreJobs)
	items, nameMap := createItemsAndNameMap(filteredPodMems,m.MinSize)

	// fmt.Printf("ITEMS: %+v\n",items)
	capacity := int(request.Amount)
      	_,_,bestConfig := algorithms.KnapsackBruteForce(capacity, items, []int{}, 0, 0, 0.)
	bestConfig = removeDuplicateInt(bestConfig)
	if len(bestConfig) == 0 {
		return nil, errors.New("no migration pod found. Is smallest pod bigger than requested amount?")
	}
	// log.Print("migrator optimal config: ",bestConfig)
	// fmt.Println("MAP",nameMap)
	// _,bestConfig := algorithms.KnapsackDynamicWeight(capacity, items,)
	
	migrations := make([]migration.MigrationCmd, 0, len(bestConfig))
	for _, idx := range bestConfig {
		pod := nameMap[idx] 
		migrations = append(migrations, migration.MigrationCmd{Pod: pod, Usage: podMems[pod]})
	}
	return migrations, nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func createItemsAndNameMap(podMems PodMemMap,minSize float64) ([]algorithms.FItem, map[int]string) {
	items := make([]algorithms.FItem, 0, len(podMems))

	nameMap := make(map[int]string)
	for name, usage := range podMems {
		if usage > minSize {
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

func (m MaxMigrator) StartMigration(cmd migration.MigrationCmd,now clock.Clock) {
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

func (m BigEnoughMigrator) StartMigration(cmd migration.MigrationCmd,now clock.Clock) {
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
