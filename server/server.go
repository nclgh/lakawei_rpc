package server

import (
	"net"
)

type Server struct {
	lis *net.TCPListener
}
