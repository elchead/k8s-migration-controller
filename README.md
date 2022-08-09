# Kubernetes migration controller

The controller is intended to migrate stateful workloads in a Kubernetes cluster. This can be useful for resilience to failure and to improve resource utilization through speculative scheduling.
The controller observes the node memory on the cluster (implemented for InfluxDB).
Because container migration is not supported by Kubernetes at the moment, modifications are necessary and a tutorial on how I achieved this can be found [here](https://astobbe.me/posts/pod-migration/).

An architecture flow chart is shown below. The unscheduler component is included as seperate component in the migration simulator.
![](./migration_controller.jpeg)

An extended k8s scheduler simulator to evaluate the performance of the migration controller on cluster utilization can be found [here](https://github.com/elchead/k8s-cluster-simulator).
An example with end-to-end integration of the controller inside the scheduler simulator can be found here #TODO

A more comprehensive motivation and explaination of the controller can be found in my [Master's thesis]() #TODO.

## Run

To use the implemented InfluxDB metric client, create a `.env` file with `INFLUXDB_TOKEN=$TOKEN`. A tutorial on how to install the Influxdb with the metrics agent including the necessary podmetrics is available [here](https://astobbe.me/posts/k8s-monitoring-with-influx-telegraf/).

The controller issues migration commands, that need to be translated to the interface of how migration is achieved. This implementation is designed to work with the Kubernetes cluster setup referenced above.
The script directory inside `pkg/migration/migration.go` needs to be modified. Sample scripts + yaml manifests are located in the same directory.

Then, simply run the client:
`go run github.com/elchead/k8s-migration-controller/cmd`

## Architecture

#TODO pic
