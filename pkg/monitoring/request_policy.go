package monitoring

type RequestPolicy interface {
	GetNodeFreeGbRequests() (criticalNodes []NodeFreeGbRequest)
	ValidateMigrationsTo(originalNode string, migratedMemory float64) string
}

type ThresholdPolicy struct {
	ThresholdFreePercent float64
	Cluster              Cluster
	Client               Clienter
}

func NewThresholdPolicyWithCluster(percent float64, cluster Cluster, client Clienter) *ThresholdPolicy {
	return &ThresholdPolicy{percent, cluster, client}
}

type NodeFreeGbRequest struct {
	Node   string
	Amount float64
}

func (t ThresholdPolicy) GetNodeFreeGbRequests() (criticalNodes []NodeFreeGbRequest) {
	nodes, _ := t.Client.GetFreeMemoryOfNodes()
	for node, availablePercent := range nodes {
		if availablePercent < t.ThresholdFreePercent {
			criticalNodes = append(criticalNodes, NodeFreeGbRequest{Node: node, Amount: t.getFreeGbAmount(availablePercent)})
		}
	}
	return criticalNodes
}

func (t ThresholdPolicy) getFreeGbAmount(availablePercent float64) float64 {
	targetAvailableGb := t.ThresholdFreePercent / 100. * t.Cluster.NodeGb
	availableGb := t.Cluster.getAvailableGb(availablePercent)
	return targetAvailableGb - availableGb

}

func (c ThresholdPolicy) enoughSpaceAvailableOn(originalNode string, podMemory float64, nodes NodeFreeMemMap) string {
	for node, freePercent := range nodes {
		if node != originalNode {
			freeGb := c.Cluster.getAvailableGb(freePercent)
			newFreeGb := freeGb - podMemory
			if c.Cluster.getFreePercent(newFreeGb) > c.ThresholdFreePercent {
				return node
			}
		}
	}
	return ""
}

func (c ThresholdPolicy) ValidateMigrationsTo(originalNode string, migratedMemory float64) string {
	nodes, _ := c.Client.GetFreeMemoryOfNodes()
	return c.enoughSpaceAvailableOn(originalNode, migratedMemory, nodes)
}
