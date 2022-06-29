package monitoring

import (
	"math"
	"time"

	"github.com/containerd/containerd/log"
	"github.com/elchead/k8s-cluster-simulator/pkg/clock"
)


const MigrationTime = 5 * time.Minute
const BackoffInterval = 45 * time.Second

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
	m.migrationFinish[pod] = m.getLastMigrationFinishTime(gbSize,t)
}

func (m *concurrentMigrationChecker) getLastMigrationFinishTime(gbSize float64,now clock.Clock) clock.Clock {
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
	adapter MigrationChecker
	migrationFinishTime []clock.Clock // TODO also for concurrent
	migrationPods []string
}

func NewBlockingMigrationChecker() *blockingMigrationChecker {
	return &blockingMigrationChecker{adapter:MigrationChecker{}}
}

func (m *blockingMigrationChecker) GetMigratingJobs(t clock.Clock) []string {
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
	m.adapter.StartMigrationWithSize(t,gbSize)	
}

func (m *blockingMigrationChecker) GetMigrationFinishTime(pod string) clock.Clock {
	return m.adapter.GetMigrationFinishTime()
} 

func (m *blockingMigrationChecker) IsReady(current clock.Clock) bool { return m.adapter.IsReady(current) }


type MigrationCheckerI interface {
	StartMigration(t clock.Clock,gbSize float64,pod string)
	GetMigrationFinishTime(pod string) clock.Clock
	IsReady(current clock.Clock) bool
}


type MigrationChecker struct {
	migrationStart clock.Clock
	migrationDuration time.Duration
}

func (m *MigrationChecker) StartMigration(t clock.Clock) {
	m.migrationStart = t
	m.migrationDuration = MigrationTime
}

func (m *MigrationChecker) StartMigrationWithSize(t clock.Clock,gbSize float64)  {
	m.migrationStart = t
	m.migrationDuration = GetMigrationTime(gbSize)
}

func (m *MigrationChecker) GetMigrationFinishTime() clock.Clock {
	return m.migrationStart.Add(m.migrationDuration)
}

func (m *MigrationChecker) IsReady(current clock.Clock) bool { return m.GetMigrationFinishTime().Add(BackoffInterval).BeforeOrEqual(current) } 
