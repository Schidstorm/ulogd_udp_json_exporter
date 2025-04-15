package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceByPort(t *testing.T) {
	service := getServiceByPort(67, "udp")
	assert.Equal(t, "bootps", service)
}
