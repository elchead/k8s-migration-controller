package migration

func main() {
	pod := "o10n-worker-m-hgvh5-hjjbm"
	Migrate([]MigrationCmd{{Pod:pod,NewNode:"zone2"}},"playground")
}
