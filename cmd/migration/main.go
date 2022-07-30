package main

import (
	"os"

	"github.com/elchead/k8s-migration-controller/pkg/migration"
)

func main() {
	pod := os.Args[1]
	migration.Migrate([]migration.MigrationCmd{{Pod:pod,NewNode:"zone2"}},"playground")
}
