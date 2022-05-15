package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetToken(t *testing.T) {
	assert.NotEqual(t, "", os.Getenv("INFLUXDB_TOKEN"))
}
