package migration

import "github.com/coredns/corefile-migration/migration"

func main() {
	pod := "o10n-worker-m-hgvh5-hjjbm"
	migration.Migrate([]MigrationCmd{{Pod:pod,NewNode:"zone2",}})
}
