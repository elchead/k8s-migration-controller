package monitoring_test

import (
	"testing"
	"time"

	"github.com/elchead/k8s-migration-controller/pkg/clock"
	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
)
const MigrationTime = 5 * time.Minute

var now = time.Now()
var clockNow = clock.NewClock(now)

func TestBlockingChecker(t *testing.T) {
	sut := monitoring.NewBlockingMigrationChecker()
	t.Run("not ready during migration", func(t *testing.T){
		sut.StartMigration(clockNow,10.,"pod1")
		assert.False(t,sut.IsReady(clockNow.Add(monitoring.BackoffInterval)))
	})
	t.Run("second migration starts after first finishes", func(t *testing.T){
		finishTimePod1 := sut.GetMigrationFinishTime("pod1")
		migrationTimePod1 := finishTimePod1.Sub(clockNow)
		sut.StartMigration(clockNow,20.,"pod2")
		assertTimeRoughlyEqual(t,finishTimePod1.Add(2*migrationTimePod1),sut.GetMigrationFinishTime("pod2"))
		assert.False(t,sut.IsReady(clockNow.Add(monitoring.BackoffInterval)))
	})
	t.Run("not ready before backoff", func(t *testing.T){
		assert.False(t,sut.IsReady(clockNow.Add(1*time.Second)))
	})
	t.Run("ready after backoff", func(t *testing.T){
		assert.True(t,sut.IsReady(clockNow.Add(MigrationTime + monitoring.BackoffInterval)))
	})
}

func TestCheckerConcurrentMigration(t *testing.T) {
	sut := monitoring.NewConcurrentMigrationChecker()
	now := clock.NewClock(time.Now())
	sut.StartMigration(now,10.,"pod1")
	assert.True(t,sut.IsReady(now.Add(1* time.Second)))
	end := sut.GetMigrationFinishTime("pod1")
	migrationTime := end.Sub(now)
	sut.StartMigration(now,20.,"pod2")
	assertTimeRoughlyEqual(t,now.Add(3*migrationTime),sut.GetMigrationFinishTime("pod2"))
	t.Run("much later migration starts much later",func(t *testing.T){
		later := now.Add(5*time.Hour)
		sut.StartMigration(later,10.,"pod3")
		assertTimeRoughlyEqual(t,later.Add(1*migrationTime),sut.GetMigrationFinishTime("pod3"))
	})
}

func assertTimeRoughlyEqual(t testing.TB,time1 clock.Clock, time2 clock.Clock) {
	assert.Equal(t,time1.ToMetaV1().Time.Round(1*time.Second),time2.ToMetaV1().Time.Round(1*time.Second))	
}
