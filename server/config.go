package server

import (
	"os"
)

const (
	RpcServiceName = "SERVICE_NAME"
	RpcServiceAddr = "SERVICE_ADDR"
	RpcServicePort = "SERVICE_PORT"
)

var (
	ServiceName string
	ServiceAddr string
	ServicePort string
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
}

func checkConfig() {
	if ServiceName == "" || ServiceAddr == "" || ServicePort == "" {
		panic("rpc config load miss")
	}
}
