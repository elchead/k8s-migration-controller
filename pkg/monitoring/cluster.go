package monitoring

type Cluster struct {
	NbrNodes int
	NodeGb   float64
}

func NewCluster() Cluster {
	return Cluster{NbrNodes: 2, NodeGb: 450.}
}

func (c Cluster) isSpaceAvailableBefore(nodes []NodeFreeGbRequest) bool {
	return len(nodes) != c.NbrNodes
}

func (c Cluster) getFreePercent(freeNodeGb float64) float64 {
	return freeNodeGb / c.NodeGb * 100.
}

func (c Cluster) getAvailableGb(freeNodePercent float64) float64 {
	return freeNodePercent / 100. * c.NodeGb
}
