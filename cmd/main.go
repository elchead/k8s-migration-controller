package main

import (
	"log"
	"os"
	"time"

	"github.com/elchead/k8s-migration-controller/pkg/clock"
	"github.com/elchead/k8s-migration-controller/pkg/migration"
	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/joho/godotenv"
)

var token string

func init() {

	err := godotenv.Load("/home/adrian/.env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}
	token = os.Getenv("INFLUXDB_TOKEN")
}

func main() {
	url := "https://westeurope-1.azure.cloud2.influxdata.com"
	org := "stobbe.adrian@gmail.com"
	client := monitoring.NewWithTime(url, token, org, "default", "-2m")
	namespace := "playground"
	cluster := monitoring.NewCluster()
	requestPolicy := monitoring.NewThresholdPolicyWithCluster(99., cluster, client)
	migrationPolicy := monitoring.NewMigrationPolicy("slope",cluster,client)
	ctrl := monitoring.NewController(requestPolicy, migrationPolicy)

	ticker := time.NewTicker(3 * time.Second)
	// checker := monitoring.NewMigrationChecker("blocking")
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			// if checker.IsReady(clock.NewClock(time.Now())) {
			migs, _ := ctrl.GetMigrations(clock.NewClock(time.Now()))
			migration.Migrate(migs, namespace)
			// }
		case <-quit:
			ticker.Stop()
			return
		}
	}
}
