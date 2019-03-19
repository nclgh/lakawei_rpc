package server

import (
	"os"
)

const (
	RpcServiceName    = "SERVICE_NAME"
	RpcServiceAddr    = "SERVICE_ADDR"
	RpcServicePort    = "SERVICE_PORT"
	RpcServiceMountIp = "SERVICE_MOUNT_IP"
)

var (
	ServiceName    string
	ServiceAddr    string
	ServicePort    string
	ServiceMountIp string
)

func initConfigFromENV() {
	if v := os.Getenv(RpcServiceName); v != "" {
		ServiceName = v
	}

	if v := os.Getenv(RpcServiceAddr); v != "" {
		ServiceAddr = v
	}

	if v := os.Getenv(RpcServicePort); v != "" {
		ServicePort = v
	}

	if v := os.Getenv(RpcServiceMountIp); v != "" {
		ServiceMountIp = v
	}
}

func initConfigFromFile() {
	c := GetConfig()

	if v := c.DefaultString("ServiceName", ""); v != "" {
		ServiceName = v
	}

	if v := c.DefaultString("ServiceAddr", ""); v != "" {
		ServiceAddr = v
	}

	if v := c.DefaultString("ServicePort", ""); v != "" {
		ServicePort = v
	}

	if v := c.DefaultString("ServiceMountIp", ""); v != "" {
		ServiceMountIp = v
	}
}

func checkConfig() {
	if ServiceName == "" || ServiceAddr == "" || ServicePort == "" || ServiceMountIp == "" {
		panic("rpc config load miss")
	}
}
