package monitoring

import (
	"fmt"
	"log"

	"github.com/elchead/k8s-cluster-simulator/pkg/clock"
	"github.com/elchead/k8s-migration-controller/pkg/migration"
	"github.com/pkg/errors"
)

type ControllerI interface {
	GetMigrations(t clock.Clock) (migrations []migration.MigrationCmd, err error)
}

type Controller struct {
	Requester RequestPolicy
	Migrator  MigrationPolicyNew
	MinRequestSize float64
}

// TODO remove
func NewControllerWithPolicy(policy *ThresholdPolicy) *Controller {
	return NewController(policy,  NewMigrationPolicy("max", policy.Cluster,policy.Client))
}

func NewController(requester RequestPolicy, migrater MigrationPolicyNew) *Controller {
	return &Controller{Requester:requester, Migrator:migrater,MinRequestSize:7.}
}

type NodeFullError struct{
	Request NodeFreeGbRequest
	Migrations []migration.MigrationCmd
}

func (m *NodeFullError) Error() string {
	return fmt.Sprintf("Migration of pods: %v failed because nodes are full. no place to fullfill request %v",m.Migrations,m.Request)
}

func (c Controller) GetMigrations(t clock.Clock) (migrations []migration.MigrationCmd, err error) {
	nodeFreeRequests := c.Requester.GetNodeFreeGbRequests()
	for _, request := range nodeFreeRequests {
		if request.Amount < c.MinRequestSize {
			log.Printf("migrator request too small, ignoring %v", request.Amount)
			return nil, nil
		}		
		log.Printf("migrator requesting: %v\n", request)
		cmds, err := c.Migrator.GetMigrationCmds(t,request)
		if err != nil {
			return nil,errors.Wrap(err, "problem during migration request")
		}
		validatedCmds := c.Requester.ValidateCmds(request.Node,cmds)
		if len(validatedCmds) == 0 && len(cmds) > 0 {
			return migrations, &NodeFullError{request,cmds} 
		}

		for i,_ := range migrations {
			c.Migrator.StartMigration(&migrations[i],t)

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
