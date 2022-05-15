package monitoring

import (
	"context"
	"errors"
	"fmt"
	"math"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

const memoryMetric = "memory_working_set_bytes" // "memory_usage_bytes"

type Clienter interface {
	GetPodMemories(nodeName string) (PodMemMap, error)
	GetFreeMemoryNode(nodeName string) (float64, error)
	GetFreeMemoryOfNodes() (NodeFreeMemMap, error)
	// GetPodMemorySlope(podName, time, slopeWindow string) (float64, error)

}

type Client struct {
	client    influxdb2.Client
	queryAPI  api.QueryAPI
	bucket    string
	TimeFrame string // time interval of data to query: e.g. "-5m"
}

func New(serviceUrl, token, org, bucket string) *Client {
	return NewWithTime(serviceUrl, token, org, bucket, "-20m")
}

func NewWithTime(serviceUrl, token, org, bucket, time string) *Client {
	client := influxdb2.NewClientWithOptions(serviceUrl, token, influxdb2.DefaultOptions())
	return &Client{client, client.QueryAPI(org), bucket, time}
}

func (c *Client) Query(query string) (*api.QueryTableResult, error) {
	return c.queryAPI.Query(context.Background(), query)
}

type PodMemMap map[string]float64

func (c *Client) GetPodMemoriesFromContainer(nodeName, containerName string) (PodMemMap, error) {
	query := fmt.Sprintf(`from(bucket: "%s") 
	|> range(start: %s)
	|> filter(fn: (r) => r["_measurement"] == "kubernetes_pod_container")
	|> filter(fn: (r) => r["_field"] == "%s")
	|> filter(fn: (r) => r["container_name"] == "%s")
	|> filter(fn: (r) => r["host"] == "%s")
	|> last()`, c.bucket, c.TimeFrame, memoryMetric, containerName, nodeName)
	res, err := c.Query(query) // default container: worker
	mp := make(PodMemMap)
	for err == nil && res.Next() {
		table := res.Record()
		pod := table.ValueByKey("pod_name").(string)
		mem := table.Value().(int64)
		mp[pod] = ConvertToGb(mem)
	}
	return mp, err
}

func (c *Client) GetPodMemories(nodeName string) (PodMemMap, error) {
	return c.GetPodMemoriesFromContainer(nodeName, "worker")
}

func (c *Client) GetPodMemorySlope(podName, time, slopeWindow string) (float64, error) {
	return c.GetPodMemorySlopeFromContainer(podName, "worker", time, slopeWindow)
}

func (c *Client) GetPodMemorySlopeFromContainer(podName, containerName, time, slopeWindow string) (float64, error) {
	query := fmt.Sprintf(`import "experimental/aggregate" from(bucket: "%s") 
  |> range(start: %s)
  |> filter(fn: (r) => r["_measurement"] == "kubernetes_pod_container")
  |> filter(fn: (r) => r["_field"] == "%s")
  |> filter(fn: (r) => r["pod_name"] == "%s")
  |> filter(fn: (r) => r["container_name"] == "%s")
  |> aggregate.rate(every: %s, unit: 1m, groupColumns: ["tag1", "tag2"])
  |> mean()`, c.bucket, time, memoryMetric, podName, containerName, slopeWindow)
	res, err := c.Query(query)
	if res.Next() && err == nil {
		num := res.Record().Value()
		if val, ok := num.(float64); ok {
			return val, nil
		} else {
			return -1., errors.New("conversion error")
		}
	}
	return -1., err
}

func (c *Client) GetFreeMemoryNode(nodeName string) (float64, error) {
	query := fmt.Sprintf(`from(bucket: "%s")
	|> range(start: %s)
	|> filter(fn: (r) => r["_measurement"] == "mem")
	|> filter(fn: (r) => r["_field"] == "available_percent")
	|> filter(fn: (r) => r["host"] == "%s")
	|> last()`, c.bucket, c.TimeFrame, nodeName)
	res, err := c.Query(query)
	if err == nil && res.Next() {
		num := res.Record().Value()
		if val, ok := num.(float64); ok {
			return val, nil
		}
	}
	return -1., err
}

type NodeFreeMemMap map[string]float64

func (c *Client) GetFreeMemoryOfNodes() (NodeFreeMemMap, error) {
	query := fmt.Sprintf(`from(bucket: "%s")
	|> range(start: %s)
	|> filter(fn: (r) => r["_measurement"] == "mem")
	|> filter(fn: (r) => r["_field"] == "available_percent")
	|> last()`, c.bucket, c.TimeFrame)
	res, err := c.Query(query)

	mp := make(NodeFreeMemMap)
	for err == nil && res.Next() {
		table := res.Record()
		node := table.ValueByKey("host").(string)
		available_percent := table.Value().(float64)
		mp[node] = available_percent
		fmt.Println("Free memory of", node, available_percent, "%")
	}
	return mp, err
}

func ConvertToGb(bytesSize int64) float64 {
	res := float64(bytesSize) / (1 << 30)
	return round(res, .5, 2)
}

func round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}
