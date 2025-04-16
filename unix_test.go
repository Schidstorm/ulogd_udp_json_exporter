package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceByPorttUncached(t *testing.T) {
	service := getServiceByPortUncached(67, "udp")
	assert.Equal(t, "bootps", service)
}

func TestGetServiceByPorttUncachedUnknown(t *testing.T) {
	service := getServiceByPortUncached(35445, "udp")
	assert.Equal(t, "35445", service)
}

func BenchmarkGetServiceByPortUncached(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getServiceByPortUncached(67, "udp")
	}
}

func TestGetProtoByNumberUncached(t *testing.T) {
	proto := getProtoByNumberUncached(17)
	assert.Equal(t, "udp", proto)

	proto = getProtoByNumberUncached(6)
	assert.Equal(t, "tcp", proto)

	proto = getProtoByNumberUncached(0)
	assert.Equal(t, "ip", proto)
}

func TestGetProtoByNumberUncachedUnknown(t *testing.T) {
	proto := getProtoByNumberUncached(9999)
	assert.Equal(t, "9999", proto)

	proto = getProtoByNumberUncached(-1)
	assert.Equal(t, "-1", proto)

	proto = getProtoByNumberUncached(0)
	assert.Equal(t, "ip", proto)
}

func BenchmarkGetProtoByNumberUncached(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getProtoByNumberUncached(17)
	}
}

func TestGetProtoAndService(t *testing.T) {
	protoName, serviceName := GetProtoAndService(67, 17)
	assert.Equal(t, "udp", protoName)
	assert.Equal(t, "bootps", serviceName)

	protoName, serviceName = GetProtoAndService(53, 6)
	assert.Equal(t, "tcp", protoName)
	assert.Equal(t, "domain", serviceName)

	protoName, serviceName = GetProtoAndService(53, 17)
	assert.Equal(t, "udp", protoName)
	assert.Equal(t, "domain", serviceName)
}
func BenchmarkGetProtoAndService(b *testing.B) {
	udpPorts := []int{
		67,
		53,
		7,
		19,
		21,
		37,
	}

	for i := 0; i < b.N; i++ {
		GetProtoAndService(int32(udpPorts[i%len(udpPorts)]), 17)
	}
}
