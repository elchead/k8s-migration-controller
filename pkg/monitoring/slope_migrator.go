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

	pq := make(PriorityQueue, 0)
	lenHeap := 0
	heap.Init(&pq)
	for podName := range podmems {
		slope, err := m.Client.GetPodMemorySlope(request.Node,podName,"","")
		if err != nil {
			log.L.Info("Error getting slope for pod: ",podName)
			continue
		}
		if slope > 0. {
			pq.Push(&Item{
				Name: podName,
				Priority: slope,
				Index:    lenHeap,
			})
			lenHeap++
			
			predictedUsage += slope * m.TimeAhead
		}
	}
	originalPredictedUsage := predictedUsage
	for predictedUsage > buffer  {
		if len(migrations) == lenHeap {
			log.L.Infof("cannot free up buffer (%.1f, node perc %f) by migrating all pods (predicted usage: %.1f)", buffer,nodeMem,originalPredictedUsage)
			return migrations, nil
		}
		item := heap.Pop(&pq).(*Item)
		predictedUsage -= item.Priority * m.TimeAhead
		log.L.Debug("Migrating pod: ",item.Name,", slope: ",item.Priority)
		migrations = append(migrations, migration.MigrationCmd{Pod:item.Name,Usage:podmems[item.Name]})
	}
	return 
}

// An Item is something we manage in a priority queue.
type Item struct {
	Name    string // The value of the item; arbitrary.
	Priority float64 // The priorityint    // The priority of the item in the queue.
	// The Index is needed by update and is maintained by the heap.Interface methods.
	Index int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	if n == 0 {
		return nil
	}
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}
