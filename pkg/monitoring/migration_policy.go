package monitoring

import (
	"errors"
	"fmt"
	"knapsack/algorithms"
	"log"
	"math"

	"github.com/elchead/k8s-cluster-simulator/pkg/clock"
	"github.com/elchead/k8s-migration-controller/pkg/migration"
)


func NewMigrationPolicy(policy string, cluster Cluster,client Clienter) MigrationPolicyNew {
	checker := NewBlockingMigrationChecker()
	return NewMigrationPolicyWithChecker(policy, cluster, client,checker)
}

func NewMigrationPolicyWithChecker(policy string, cluster Cluster,client Clienter,checker MigrationCheckerI) MigrationPolicyNew {
	client = NewFilteredClient(client)
	var migrator MigrationPolicy
	switch policy {
	case "slope":
		migrator = &SlopeMigrator{Cluster:cluster, Client:client,TimeAhead:5.} // TODO configure timeahead
	case "optimal":
		migrator = &OptimalMigrator{Client: client, Cluster: cluster,MinSize:5.,Checker:checker}
	case "max":
		migrator = &MaxMigrator{Cluster: cluster, Client: client}
	case "big-enough":
		migrator = &BigEnoughMigrator{Cluster: cluster, Client: client}
	default:
		log.Fatal("Unknown migration policy: ",policy)
		return nil
	}
	return MigratorAdapter{MigrationPolicy: migrator, Checker: checker,ClientRef: client.(*FilteredClient)}
}

type MigrationPolicy interface {
	GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error)
}

type MigrationPolicyNew interface {
	GetMigrationCmds(now clock.Clock, request NodeFreeGbRequest) ([]migration.MigrationCmd, error)
	StartMigration(*migration.MigrationCmd,clock.Clock)
}

type FilteredClient struct {
	Clienter
	jobsToIgnore []string
}

func NewFilteredClient(client Clienter) *FilteredClient {
	return &FilteredClient{Clienter:client}
}

func (c *FilteredClient) UpdateJobsToIgnore(jobsToIgnore []string) {
	c.jobsToIgnore = jobsToIgnore
}

func (c FilteredClient) GetPodMemories(node string) (PodMemMap, error) {
	res, err := c.Clienter.GetPodMemories(node)
	if err != nil {
		return nil, err
	}
	// fmt.Println("ignoring jobs",c.jobsToIgnore)
	filtered := filterPods(res, c.jobsToIgnore)
	return filtered, nil
}

type MigratorAdapter struct {
	MigrationPolicy
	Checker MigrationCheckerI
	ClientRef *FilteredClient
}

func (m MigratorAdapter) GetMigrationCmds(now clock.Clock,request NodeFreeGbRequest) ([]migration.MigrationCmd, error) {
	if !m.Checker.IsReady(now) {
		return nil, nil // No error otherwise no start of migration; errors.New("checker not ready")
	}
	jobsToIgnore := m.Checker.GetMigratingJobs(now)
	m.ClientRef.UpdateJobsToIgnore(jobsToIgnore)
	res,err := m.MigrationPolicy.GetMigrationCmds(request)
	return res,err
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

func (m MigratorAdapter) StartMigration(cmd *migration.MigrationCmd,now clock.Clock) {
	m.Checker.StartMigration(now,cmd.Usage,cmd.Pod)
	cmd.FinishAt = m.Checker.GetMigrationFinishTime(cmd.Pod)
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
	podMems, err := m.Client.GetPodMemories(request.Node)
	if err != nil {
		return nil, err
	}
	items, nameMap := createItemsAndNameMap(podMems,m.MinSize)

	// fmt.Printf("ITEMS: %+v\n",items)
	capacity := int(request.Amount)
      	_,_,bestConfig := algorithms.KnapsackBruteForce(capacity, items, []int{}, 0, 0, 0.)
	bestConfig = removeDuplicateInt(bestConfig)
	if len(bestConfig) == 0 {
		return nil, errors.New("no migration pod found. Is smallest pod bigger than requested amount?")
	}
	// log.Print("migrator optimal config: ",bestConfig)
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
