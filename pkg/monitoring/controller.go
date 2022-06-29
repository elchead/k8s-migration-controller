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
	MinRequestSize float64
}

// TODO remove
func NewControllerWithPolicy(policy *ThresholdPolicy) *Controller {
	return NewController(policy,&MaxMigrator{policy.Cluster, policy.Client})
}

func NewController(requester RequestPolicy, migrater MigrationPolicy) *Controller {
	return &Controller{Requester:requester, Migrator:migrater,MinRequestSize:7.}
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
		if request.Amount < c.MinRequestSize {
			log.Printf("migrator request too small, ignoring %v", request.Amount)
			return nil, nil
		}		
		log.Printf("migrator requesting: %v\n", request)
		cmds, err := c.Migrator.GetMigrationCmds(request)
		if err != nil {
			return nil,errors.Wrap(err, "problem during migration request")
		}
		migrationSize := sumPodMemories(cmds)
		if migratingNode :=c.Requester.ValidateMigrationsTo(request.Node, migrationSize); migratingNode != "" {
			log.Printf("migrator request fulfilled (%v Gb): %v\n",sumPodMemories(cmds), cmds)
			for i := range cmds {
				cmds[i].NewNode = migratingNode
			}
			migrations = append(migrations, cmds...)
		} else {
			return migrations, &NodeFullError{request,cmds}
		}

		for _,cmd := range cmds {
			c.Migrator.StartMigration(cmd)
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
