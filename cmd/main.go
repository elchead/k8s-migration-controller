package main

import (
	"fmt"
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
	client := monitoring.NewClientWithTime(url, token, org, "default", "-2m","-1m")
	namespace := "playground"
	cluster := monitoring.NewCluster()
	requestPolicy := monitoring.NewThresholdPolicyWithCluster(30., cluster, client)
	migrationPolicy := monitoring.NewMigrationPolicy("slope",cluster,client)
	ctrl := monitoring.NewController(requestPolicy, migrationPolicy)

	ticker := time.NewTicker(15 * time.Second)
	// checker := monitoring.NewMigrationChecker("blocking")
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			// if checker.IsReady(clock.NewClock(time.Now())) {
			migs, _ := ctrl.GetMigrations(clock.NewClock(time.Now()))
			if (len(migs) > 0) {
				fmt.Println(migs)
			}
			migration.Migrate(migs, namespace)
			// }
		case <-quit:
			ticker.Stop()
			return
		}
	}
}
