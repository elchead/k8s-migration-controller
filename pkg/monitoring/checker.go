package monitoring

import (
	"math"
	"time"

	"github.com/containerd/containerd/log"
	"github.com/elchead/k8s-cluster-simulator/pkg/clock"
)


const BackoffInterval = 30 * time.Second // at least as big as polling interval of client (to see migration update)

func GetMigrationTime(gbSz float64) time.Duration {
	return time.Duration(math.Ceil(3.3506*gbSz))*time.Second
}

func NewConcurrentMigrationChecker() *concurrentMigrationChecker {
	return &concurrentMigrationChecker{make(map[string]clock.Clock),make(map[string]clock.Clock),nil}
}

type concurrentMigrationChecker struct {
	migrationFinish map[string]clock.Clock
	migrationStart map[string]clock.Clock
	latestFinish *clock.Clock
}
func (m *concurrentMigrationChecker) StartMigration(t clock.Clock,gbSize float64,pod string) {
	m.migrationStart[pod] = t
	m.migrationFinish[pod] = m.updateLastMigrationFinishTime(gbSize,t)
}

func (m concurrentMigrationChecker) GetMigratingJobs(now clock.Clock) []string { 
	var res []string
	for pod,finishTime := range m.migrationFinish {
		if !finishTime.BeforeOrEqual(now) {
			res = append(res,pod)
		}
	}
	return res
}

func (m *concurrentMigrationChecker) updateLastMigrationFinishTime(gbSize float64,now clock.Clock) clock.Clock {
	if m.latestFinish == nil {
		m.latestFinish = &now
	}

	startTime := maxClock(*m.latestFinish, now)
	res := startTime.Add(GetMigrationTime(gbSize))
	m.latestFinish = &res
	return res
}

func maxClock(t1, t2 clock.Clock) (startTime clock.Clock) {
	if t2.BeforeOrEqual(t1) {
		startTime = t1
	} else {
		startTime = t2
	}
	return
}

func (m *concurrentMigrationChecker) GetMigrationFinishTime(pod string) clock.Clock {
	return m.migrationFinish[pod]
} 

func (m *concurrentMigrationChecker) IsReady(current clock.Clock) bool { return true }

func NewMigrationChecker(checkerType string) MigrationCheckerI {
	switch checkerType {
	case "blocking":  return NewBlockingMigrationChecker()
	case "concurrent": return NewConcurrentMigrationChecker()
	default: 
		log.L.Warnf("unsupported checker type %v; using blocking type", checkerType) 
		return NewBlockingMigrationChecker()
	}
}
type blockingMigrationChecker struct {
	wrapper *concurrentMigrationChecker
	migrationFinishTime []clock.Clock // TODO also for concurrent
	migrationPods []string
}

func NewBlockingMigrationChecker() *blockingMigrationChecker {
	return &blockingMigrationChecker{wrapper:NewConcurrentMigrationChecker()}
}

func (m *blockingMigrationChecker) GetMigratingJobs(t clock.Clock) []string { // TODO use concurrentMigrationChecker
	var res []string
	for idx,v := range m.migrationFinishTime {
		if !v.BeforeOrEqual(t) {
			res = append(res,m.migrationPods[idx])
		}
	}
	m.migrationPods = res
	return res
}

func (m *blockingMigrationChecker) StartMigration(t clock.Clock,gbSize float64,pod string) {
	m.wrapper.StartMigration(t,gbSize,pod)
}

func (m *blockingMigrationChecker) GetMigrationFinishTime(pod string) clock.Clock {
	return m.wrapper.GetMigrationFinishTime(pod)
} 

func (m *blockingMigrationChecker) IsReady(current clock.Clock) bool { 
 	if m.wrapper.latestFinish == nil  { 
		return true 
	} else { 
		return m.wrapper.latestFinish.Add(BackoffInterval).BeforeOrEqual(current)
	}
}



type MigrationCheckerI interface {
	StartMigration(t clock.Clock,gbSize float64,pod string)
	GetMigrationFinishTime(pod string) clock.Clock
	IsReady(current clock.Clock) bool
	GetMigratingJobs(t clock.Clock) []string
}
