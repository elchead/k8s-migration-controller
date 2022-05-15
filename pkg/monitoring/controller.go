package monitoring

import (
	"log"

	"github.com/elchead/k8s-migration-controller/pkg/migration"
)

type Controller struct {
	Requester RequestPolicy
	Migrator  MigrationPolicy
}

// TODO remove
func NewControllerWithPolicy(policy *ThresholdPolicy) *Controller {
	return &Controller{policy, &MaxMigrator{policy.Cluster, policy.Client}}
}

func NewController(requester RequestPolicy, migrater MigrationPolicy) *Controller {
	return &Controller{requester, migrater}
}

type NodeFullError struct{}

func (m *NodeFullError) Error() string {
	return "nodes are full. no place to migrate"
}

func (c Controller) GetMigrations() (migrations []migration.MigrationCmd, err error) {
	nodeFreeRequests := c.Requester.GetNodeFreeGbRequests()
	for _, request := range nodeFreeRequests {
		cmds, err := c.Migrator.GetMigrationCmds(request)
		if err != nil {
			log.Println("Problem during migration request:", err)
		}
		if c.Requester.ValidateMigrationsTo(request.Node, sumPodMemories(cmds)) != "" {
			// TODO use node name to migrate to
			migrations = append(migrations, cmds...)
		} else {
			return migrations, &NodeFullError{}
		}
	}
	return migrations, nil
}

func sumPodMemories(cmds []migration.MigrationCmd) float64 {
	memSum := 0.
	for _, cmd := range cmds {
		memSum += cmd.Usage
	}
	return memSum
}
