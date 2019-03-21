package client

import (
	"fmt"
	"sync"
	"time"
	"net/rpc"
	"math/rand"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/nclgh/lakawei_discover"
	"github.com/nclgh/lakawei_rpc/helper"
)

var (
	ErrRpcCallTimeout   = errors.New("rpc call timeout")
	ErrRemoteNoInstance = errors.New("remote no instance")
)

const (
	MaxRequestTime        = 10000 * time.Millisecond
	DefaultMaxRequestTime = 5000 * time.Millisecond
	DefaultRetryTime      = 5
	AddrUpdateTime        = 2 * time.Second
)

type RpcClient struct {
	ServiceName string
	clients     map[string]*rpc.Client
	RAddr       []string
	UpdateTime  int64
	lock        sync.RWMutex
	once        sync.Once
	freshLock   sync.Mutex
}

type RpcRequestCtx struct {
	MaxRequestTime int64 // Millisecond
	RetryTime      int
}

func (ctx *RpcRequestCtx) GetMaxRequestTime() time.Duration {
	if ctx.MaxRequestTime <= 0 {
		return DefaultMaxRequestTime
	}
	if int64(ctx.MaxRequestTime) > int64(MaxRequestTime) {
		return MaxRequestTime
	}
	return time.Duration(ctx.MaxRequestTime * int64(time.Millisecond))
}

func (ctx *RpcRequestCtx) GetRetryTime() int {
	if ctx.RetryTime <= 0 {
		return DefaultRetryTime
	}
	return ctx.RetryTime
}

func InitClient(serviceName string) (*RpcClient, error) {
	cli := &RpcClient{}
	cli.ServiceName = serviceName
	cli.clients = make(map[string]*rpc.Client)
	st := time.Now()
	cli.once.Do(cli.freshRemoteAddr)
	//time.Sleep(time.Second)
	et := time.Now()
	sub := et.Sub(st).Nanoseconds()
	fmt.Println("first connect cost", sub, "ns")
	if len(cli.clients) <= 0 {
		logrus.Errorf("service %s has no instance", cli.ServiceName)
	}

	go func() {
		for {
			cli.freshRemoteAddr()
			time.Sleep(AddrUpdateTime)
		}
	}()
	return cli, nil
}

func (cli *RpcClient) Call(ctx *RpcRequestCtx, method string, req interface{}, ret interface{}) error {
	cli.once.Do(func() {
		panic("rpc addr not init")
	})
	expireCh := make(chan error, 1)
	doneCh := make(chan interface{}, 0)

	go func() {
		var err error
		for i := 0; i < ctx.GetRetryTime(); i++ {
			rpcCli, err1 := cli.getRpcClient()
			if err1 != nil {
				err = err1
				continue
			}
			methodRouter := fmt.Sprintf("%s.%s", cli.ServiceName, method)
			err = rpcCli.Call(methodRouter, req, ret)
			if err != nil {
				logrus.Warnf("call cli.ServiceName method failed at %v err: %v", i, err)
				continue
			}
			doneCh <- ret
			return
		}
		logrus.Errorf("call cli.ServiceName method failed err: %v", err)
		doneCh <- err
	}()

	go func() {
		time.Sleep(ctx.GetMaxRequestTime())
		expireCh <- ErrRpcCallTimeout
	}()

	for {
		select {
		case timeout := <-expireCh:
			return timeout
		case done := <-doneCh:
			if err, ok := done.(error); ok {
				return err
			}
			return nil
		}
	}
}

func (cli *RpcClient) getRpcClient() (*rpc.Client, error) {
	cli.lock.RLock()
	defer cli.lock.RUnlock()
	addrNum := len(cli.RAddr)
	if addrNum <= 0 {
		return nil, ErrRemoteNoInstance
	}
	selectedAddr := cli.RAddr[rand.Int()%addrNum]
	return cli.clients[selectedAddr], nil
}

func (cli *RpcClient) freshRemoteAddr() {
	cli.freshLock.Lock()
	defer cli.freshLock.Unlock()
	waitNum := 2
	waitCh := make(chan bool, waitNum)
	addrs := lakawei_discover.GetServiceAddr(cli.ServiceName)
	addrMap := make(map[string]bool)
	for _, addr := range addrs {
		addrMap[addr] = true
	}
	// 删除没在服务发现地址的client
	go func() {
		cli.lock.Lock()
		defer cli.lock.Unlock()
		defer func() {
			waitCh <- true
		}()
		aliveAddr := make([]string, 0)
		toDeleteClient := make([]*rpc.Client, 0)
		for _, addr := range cli.RAddr {
			if !addrMap[addr] {
				toDeleteClient = append(toDeleteClient, cli.clients[addr])
				delete(cli.clients, addr)
				continue
			}
			aliveAddr = append(aliveAddr, addr)
		}
		cli.RAddr = aliveAddr

		go func() {
			defer helper.RecoverPanic(func(err interface{}, stacks string) {
				logrus.Errorf("close dead client panic: %v, stack: %v", err, stacks)
			})
			time.Sleep(MaxRequestTime)
			for _, c := range toDeleteClient {
				c.Close()
			}
		}()
	}()

	// 增加新的client
	go func() {
		cli.lock.RLock()
		defer cli.lock.RUnlock()
		notExistAddr := make([]string, 0)
		for _, addr := range addrs {
			if _, ok := cli.clients[addr]; !ok {
				notExistAddr = append(notExistAddr, addr)
			}
		}
		go func() {
			defer func() {
				waitCh <- true
			}()
			defer helper.RecoverPanic(func(err interface{}, stacks string) {
				logrus.Errorf("add new client panic: %v, stack: %v", err, stacks)
			})
			clis := make(map[string]*rpc.Client, 0)
			aliveAddr := make([]string, 0)
			for _, addr := range notExistAddr {
				c, err := rpc.Dial("tcp", addr)
				if err != nil {
					logrus.Errorf("can't connect addr: %v, err: %v", addr, err)
					continue
				}
				clis[addr] = c
				aliveAddr = append(aliveAddr, addr)
			}
			if len(clis) <= 0 {
				return
			}
			cli.lock.Lock()
			defer cli.lock.Unlock()
			cli.RAddr = append(cli.RAddr, aliveAddr...)
			for a, c := range clis {
				cli.clients[a] = c
			}
		}()
	}()
	for {
		select {
		case <-waitCh:
			waitNum--
			if waitNum <= 0 {
				return
			}
		}
	}
}
