package monitoring

import (
	"github.com/containerd/containerd/log"
	"github.com/elchead/k8s-migration-controller/pkg/migration"
)

type SlopeMigrator struct {
	Cluster Cluster
	Client  Clienter
	TimeAhead float64 // in min
	Buffer float64 // in Gb
}


func (m SlopeMigrator) GetMigrationCmds(request NodeFreeGbRequest) ([]migration.MigrationCmd, error) {
	podmems,err := m.Client.GetPodMemories(request.Node)

	largestSlope := 0.
	var largestSlopePod string

	predictedUsage := 0.
	for name,_:= range podmems {
		slope, err := m.Client.GetPodMemorySlope(request.Node,name,"","")

		predictedUsage += slope * m.TimeAhead
		if err != nil {
			log.L.Info("Error getting slope for pod: ",name)
		}
		if slope > largestSlope {
			largestSlope = slope
			largestSlopePod = name
		}
	}
	if largestSlopePod  == "" || predictedUsage < m.Buffer  {
		return nil,err
	}
	return []migration.MigrationCmd{{Pod:largestSlopePod,Usage:podmems[largestSlopePod]}},err
}
