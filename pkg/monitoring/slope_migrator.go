package monitoring

import (
	"container/heap"

	"github.com/containerd/containerd/log"
	"github.com/elchead/k8s-migration-controller/pkg/migration"
)

type SlopeMigrator struct {
	Cluster Cluster
	Client  Clienter
	TimeAhead float64 // in min
}



func (m SlopeMigrator) GetMigrationCmds(request NodeFreeGbRequest) (migrations []migration.MigrationCmd, err error) {

	nodeMem, _:= m.Client.GetFreeMemoryNode(request.Node)
	buffer := m.Cluster.getAvailableGb(nodeMem)


	podmems,err := m.Client.GetPodMemories(request.Node)

	predictedUsage := 0.

	pq := make(PriorityQueue, len(podmems))
	i := 0
	for name,_:= range podmems {
		slope, err := m.Client.GetPodMemorySlope(request.Node,name,"","")
		
		pq[i] = &Item{
			name: name,
			priority: slope,
			index:    i,
		}
		i++
		
		predictedUsage += slope * m.TimeAhead
		if err != nil {
			log.L.Info("Error getting slope for pod: ",name)
		}
	}
	heap.Init(&pq)
	for predictedUsage > buffer {
		item := heap.Pop(&pq).(*Item)
		predictedUsage -= item.priority * m.TimeAhead
		log.L.Info("Migrating pod: ",item.name,", slope: ",item.priority)
		migrations = append(migrations, migration.MigrationCmd{Pod:item.name,Usage:podmems[item.name]})
	}
	return 
}

// An Item is something we manage in a priority queue.
type Item struct {
	name    string // The value of the item; arbitrary.
	priority float64 // The priorityint    // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}
