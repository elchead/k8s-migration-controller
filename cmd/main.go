package main

import (
	"time"

	"github.com/elchead/k8s-migration-controller/pkg/clock"
	"github.com/elchead/k8s-migration-controller/pkg/migration"
	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
)

var token string

func init() {
	token = monitoring.ReadToken("/home/adrian/.env")
}


func main() {
	url := "https://westeurope-1.azure.cloud2.influxdata.com"
	org := "stobbe.adrian@gmail.com"
	client := monitoring.NewClientWithTime(url, token, org, "default", "-3m","-1m")
	namespace := "playground"
	cluster := monitoring.NewCluster()
	requestPolicy := monitoring.NewSingleThresholdPolicyWithCluster(30., cluster, client)
	migrationPolicy := monitoring.NewMigrationPolicy("slope",cluster,client)
	ctrl := monitoring.NewController(requestPolicy, migrationPolicy)

	ticker := time.NewTicker(15 * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			migs, _ := ctrl.GetMigrations(clock.NewClock(time.Now()))
			migration.Migrate(migs, namespace)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}
