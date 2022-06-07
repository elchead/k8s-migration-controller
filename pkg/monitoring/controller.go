package monitoring

import (
	"fmt"
	"log"

	"github.com/elchead/k8s-migration-controller/pkg/migration"
	"github.com/pkg/errors"
)

type ControllerI interface {
	GetMigrations() (migrations []migration.MigrationCmd, err error)
}

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

type NodeFullError struct{
	Request NodeFreeGbRequest
	Migrations []migration.MigrationCmd
}

func (m *NodeFullError) Error() string {
	return fmt.Sprintf("Migration of pods: %v failed because nodes are full. no place to fullfill request %v",m.Migrations,m.Request)
}

func (c Controller) GetMigrations() (migrations []migration.MigrationCmd, err error) {
	nodeFreeRequests := c.Requester.GetNodeFreeGbRequests()
	for _, request := range nodeFreeRequests {
		log.Printf("migrator requesting: %v\n", request)
		cmds, err := c.Migrator.GetMigrationCmds(request)
		if err != nil {
			return nil,errors.Wrap(err, "problem during migration request")
		}
		if c.Requester.ValidateMigrationsTo(request.Node, sumPodMemories(cmds)) != "" {
			// TODO use node name to migrate to
			log.Printf("migrator request fulfilled (%v Gb): %v\n",sumPodMemories(cmds), cmds)
			migrations = append(migrations, cmds...)
		} else {
			return migrations, &NodeFullError{request,cmds}
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
