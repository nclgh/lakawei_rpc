package server

import (
	"os"
	"net"
	"fmt"
	"net/rpc"
	"syscall"
	"os/signal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/nclgh/lakawei_discover"
)

var (
	RpcServer *Server
)

func Init() {
	// ENV > FILE
	initConfigFromFile()
	initConfigFromENV()

	// 加载完配置后检查下配置
	checkConfig()
}

func Run(rcvr interface{}) error {
	rpc.Register(rcvr)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":"+ServicePort)
	logrus.Infof("Service to listening tcp service %s port:%s", ServiceName, ServicePort)

	if err != nil {
		panic(err)
	}
	tcplisten, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		panic(err)
	}
	RpcServer = &Server{
		lis: tcplisten,
	}
	lakawei_discover.Register(ServiceName, fmt.Sprintf("%s:%s", ServiceAddr, ServicePort))
	errCh := make(chan error, 1)
	go func() {
		logrus.Infof("Start to run rpc service %s %s:%s", ServiceName, ServiceAddr, ServicePort)
		for {
			conn, err2 := RpcServer.lis.Accept()
			if err2 != nil {
				errCh <- err2
				break
			}
			//使用goroutine单独处理rpc连接请求
			go rpc.ServeConn(conn)
		}
	}()
	return waitSignal(errCh)
}

func waitSignal(errCh <-chan error) error {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	defer lakawei_discover.Unregister()
	for {
		select {
		case sig := <-ch:
			fmt.Printf("Got signal: %s, Exit..\n", sig)
			return errors.New(sig.String())
		case err := <-errCh:
			fmt.Printf("Engine run error: %s, Exit..\n", err)
			return err
		}
	}
}
