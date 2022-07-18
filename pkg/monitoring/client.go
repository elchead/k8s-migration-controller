package monitoring

import (
	"github.com/containerd/containerd/log"
)

type FilteredClient struct {
	Clienter
}

func NewFilteredClient(client Clienter) *FilteredClient {
	return &FilteredClient{Clienter:client}
}

func (c FilteredClient) GetPodMemories(node string) (PodMemMap, error) {
	res, err := c.Clienter.GetPodMemories(node)
	if err != nil {
		return nil, err
	}
	filtered := c.filterPods(res)
	return filtered, nil
}

func (c FilteredClient) filterPods(podMems PodMemMap) ( PodMemMap) {
	filteredPodMems := PodMemMap{}
	for job, amount := range podMems {
		podName := job
		if perc := c.GetRuntimePercentage(podName); perc > 95 {
			log.L.Debugf("Pod %s has more than 95 percent of runtime: %f. Ignoring for migration",podName,perc)
			continue
		}
		exec := c.GetExecutionTime(podName)
		runtime := c.GetRuntime(podName)
		// fmt.Printf("runtime %s: %d exectime %d\n",podName,runtime,exec)
	
		if runtime - exec < 180 {
			log.L.Debugf("Pod %s remaining runtime is only %d. Ignoring for migration",podName,runtime-exec)
			continue
		}
		filteredPodMems[job] = amount
	}
	return filteredPodMems
}

