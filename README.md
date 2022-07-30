## Run

First, cd into root dir of repository.

To run the controller:
`go run github.com/elchead/k8s-migration-controller/pkg/migration/main.go`

To test pod migration with specfic pod name:
`go run ./cmd/migration/main.go $POD_NAME`
