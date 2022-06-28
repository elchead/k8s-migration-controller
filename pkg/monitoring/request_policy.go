package monitoring

import (
	"fmt"
	"log"
)

type RequestPolicy interface {
	GetNodeFreeGbRequests() (criticalNodes []NodeFreeGbRequest)
	ValidateMigrationsTo(originalNode string, migratedMemory float64) string
	SetThreshold(float64)
}

type SlopeRequester struct {
	ThresholdFreePercent float64
	Cluster Cluster
	Client  Clienter
	PredictionTime float64
}

func NewSlopePolicyWithClusterAndTime(percent,predictionTime float64, cluster Cluster, client Clienter) *SlopeRequester {
	return &SlopeRequester{percent,cluster, client,predictionTime}
}


func NewSlopePolicyWithCluster(percent float64, cluster Cluster, client Clienter) *SlopeRequester {
	return &SlopeRequester{percent,cluster, client,5.}
}

func (t SlopeRequester) GetNodeFreeGbRequests() (criticalNodes []NodeFreeGbRequest) {
	nodes, _ := t.Client.GetFreeMemoryOfNodes()
	for node, availablePercent := range nodes {
		pods, _ := t.Client.GetPodMemories(node)
		slope := 0.
		for podname := range pods {
			s, _ := t.Client.GetPodMemorySlope(node, podname, "now", "1m")
			slope += s	
			
		}
		predictedUsage := slope * t.PredictionTime // min
		predictedPercent := t.Cluster.GetUsagePercent(predictedUsage)
		freePercent := availablePercent - predictedPercent
		if freePercent < t.ThresholdFreePercent {
			fmt.Println("Requester predicts usage ",predictedUsage,"GB of node",node," currently free ", t.Cluster.getAvailableGb(availablePercent),"GB ", availablePercent, " %")
			criticalNodes = append(criticalNodes, NodeFreeGbRequest{Node: node, Amount: getFreeGbAmount(t.ThresholdFreePercent,freePercent,t.Cluster)})
		}
	}
	return criticalNodes
}

func (c SlopeRequester) enoughSpaceAvailableOn(originalNode string, podMemory float64, nodeAvailablePercents NodeFreeMemMap) string {
	return getAvailableNodeWithLeastUsage(c.Cluster,c.ThresholdFreePercent, nodeAvailablePercents, originalNode, podMemory)
}

func getAvailableNodeWithLeastUsage(c Cluster, thresholdFreePercent float64,nodeAvailablePercents NodeFreeMemMap, originalNode string, podMemory float64) (string) {
	leastNode := getLeastUsedNode(nodeAvailablePercents, originalNode)
	fmt.Println("Least used node ",leastNode)
	availablePercent := nodeAvailablePercents[leastNode]
	freeGb := c.getAvailableGb(availablePercent)
	fmt.Println("Free GB on least node ",freeGb)

	newFreeGb := freeGb - podMemory
	if c.GetUsagePercent(newFreeGb) > thresholdFreePercent {
		return leastNode
	}
	log.Println("No node available with enough space (free percentage of nodes):",nodeAvailablePercents)
	return ""
}

func getLeastUsedNode(nodeAvailablePercents NodeFreeMemMap, originalNode string) string {
	mostFreePercent := 0. // c.NodeGb
	leastNode := ""
	for node, availablePercent := range nodeAvailablePercents {
		if availablePercent > mostFreePercent && node != originalNode {
			mostFreePercent = availablePercent
			leastNode = node
		}
	}
	return leastNode
}

func (c SlopeRequester) ValidateMigrationsTo(originalNode string, migratedMemory float64) string {
	nodes, _ := c.Client.GetFreeMemoryOfNodes()
	return c.enoughSpaceAvailableOn(originalNode, migratedMemory, nodes)
}

func (c *SlopeRequester) SetThreshold(thresholdPercent float64) {
	c.ThresholdFreePercent = thresholdPercent
}

type ThresholdPolicy struct {
	ThresholdFreePercent float64
	Cluster              Cluster
	Client               Clienter
}

func NewRequestPolicy(policy string, cluster Cluster,client Clienter,threshold float64) RequestPolicy {
	switch policy {
	case "slope":
		return NewSlopePolicyWithCluster(threshold, cluster, client)
	case "thresold":
		return NewThresholdPolicyWithCluster(threshold,cluster,client)
	default:
		log.Println("Defaulting to threshold request policy. Unknown policy: ",policy)
		return NewThresholdPolicyWithCluster(threshold,cluster,client)
	}
}

func NewThresholdPolicyWithCluster(percent float64, cluster Cluster, client Clienter) *ThresholdPolicy {
	return &ThresholdPolicy{percent, cluster, client}
}

type NodeFreeGbRequest struct {
	Node   string
	Amount float64
}

func (c *ThresholdPolicy) SetThreshold(thresholdPercent float64) {
	c.ThresholdFreePercent = thresholdPercent
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
	return getFreeGbAmount(t.ThresholdFreePercent,availablePercent,t.Cluster)
}

func getFreeGbAmount(thresholdPercent,availablePercent float64,cluster Cluster) float64 {
	targetAvailableGb := thresholdPercent / 100. * cluster.NodeGb
	availableGb := cluster.getAvailableGb(availablePercent)
	return targetAvailableGb - availableGb
}

func (c ThresholdPolicy) enoughSpaceAvailableOn(originalNode string, podMemory float64, nodes NodeFreeMemMap) string {
	return getAvailableNodeWithLeastUsage(c.Cluster,c.ThresholdFreePercent, nodes, originalNode, podMemory)
}

func (c ThresholdPolicy) ValidateMigrationsTo(originalNode string, migratedMemory float64) string {
	nodes, _ := c.Client.GetFreeMemoryOfNodes()
	return c.enoughSpaceAvailableOn(originalNode, migratedMemory, nodes)
}
