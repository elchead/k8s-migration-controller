package monitoring

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func ReadToken(envFile string) string {
	err := godotenv.Load(envFile)

	if err != nil {
		log.Fatal("Error loading .env file",err)
	}
	return os.Getenv("INFLUXDB_TOKEN")
}

type PodMemMap map[string]float64

type Clienter interface {
	GetPodMemories(nodeName string) (PodMemMap, error)
	GetFreeMemoryNode(nodeName string) (float64, error)
	GetFreeMemoryOfNodes() (NodeFreeMemMap, error) // in %
	GetPodMemorySlope(node,podName string) (float64, error) // in GB
}

func (original PodMemMap) Copy() PodMemMap {
	cp := make(PodMemMap)
	for k, v := range original {
		cp[k] = v
	}
	return cp
}

func (c PodMemMap) CountMigrations(name string) (int,error) { 
	_, ok := c[name]
	if !ok {
		return 0, fmt.Errorf("pod %s not found", name)
	}
	onlyMPrefix := strings.Split(name,"o")[0]
	return strings.Count(onlyMPrefix,"m"), nil
	
}
