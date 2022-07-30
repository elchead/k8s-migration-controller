package main

import "github.com/elchead/k8s-migration-controller/pkg/migration"

func main() {
	pod := "o10n-worker-m-hgvh5-hjjbm"
	migration.Migrate([]migration.MigrationCmd{{Pod:pod,NewNode:"zone2"}},"playground")
}
