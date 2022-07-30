package monitoring_test

import (
	"testing"

	"github.com/elchead/k8s-migration-controller/pkg/monitoring"
	"github.com/stretchr/testify/assert"
)

var token string

func init() {
	token = monitoring.ReadToken("../../test/token.env")
}


func TestClient(t *testing.T) {
	assert.NotEmpty(t,token)
	url := "https://westeurope-1.azure.cloud2.influxdata.com"
	org := "stobbe.adrian@gmail.com"
	client := monitoring.NewClientWithTime(url, token, org, "default", "-2m")
	mems, err := client.GetFreeMemoryOfNodes()
	assert.NoError(t, err)
	assert.Equal(t,mems,"r")

	pods, err := client.GetPodMemories("zone2")
	assert.NoError(t, err)
	assert.Equal(t,pods,"r")

	// keys := reflect.ValueOf(pods).MapKeys()
    	podName := "o10n-worker-m-58gsz-dfhfw" // keys[0].String()
	slope,err := client.GetPodMemorySlope("zone2",podName,"","")
	assert.NoError(t,err)
	assert.Equal(t,slope,"r")

}
