package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/Wregret/breeoche/rpcapi"
	"time"
)

// RPCClient is a net/rpc client for Breeoche.
type RPCClient struct {
	addr string
}

func NewRPCClient(addr string) *RPCClient {
	if addr == "" {
		addr = defaultServerAddr
	}
	return &RPCClient{addr: addr}
}

func (c *RPCClient) Ping() (string, error) {
	addr := c.addr
	for i := 0; i < maxRedirects; i++ {
		var reply rpcapi.PingReply
		if err := c.call(addr, rpcapi.KVServiceName+".Ping", rpcapi.PingArgs{}, &reply); err != nil {
			return "", err
		}
		if reply.Error != "" {
			return "", errors.New(reply.Error)
		}
		return reply.Message, nil
	}
	return "", errors.New("too many redirects")
}

func (c *RPCClient) Get(key string) (string, error) {
	addr := c.addr
	for i := 0; i < maxRedirects; i++ {
		var reply rpcapi.GetReply
		if err := c.call(addr, rpcapi.KVServiceName+".Get", rpcapi.GetArgs{Key: key}, &reply); err != nil {
			return "", err
		}
		if reply.Error == rpcapi.ErrNotLeader && reply.LeaderAddr != "" {
			addr = reply.LeaderAddr
			continue
		}
		if reply.Error != "" {
			return "", errors.New(reply.Error)
		}
		if !reply.Found {
			return "", errors.New("key not found")
		}
		return reply.Value, nil
	}
	return "", errors.New("too many redirects")
}

func (c *RPCClient) Set(key string, value string) error {
	return c.doMutate(rpcapi.KVServiceName+".Set", rpcapi.SetArgs{Key: key, Value: value})
}

func (c *RPCClient) Insert(key string, value string) error {
	return c.doMutate(rpcapi.KVServiceName+".Insert", rpcapi.InsertArgs{Key: key, Value: value})
}

func (c *RPCClient) Delete(key string) error {
	return c.doMutate(rpcapi.KVServiceName+".Delete", rpcapi.DeleteArgs{Key: key})
}

func (c *RPCClient) doMutate(method string, args interface{}) error {
	addr := c.addr
	for i := 0; i < maxRedirects; i++ {
		var reply rpcapi.MutateReply
		if err := c.call(addr, method, args, &reply); err != nil {
			return err
		}
		if reply.Error == rpcapi.ErrNotLeader && reply.LeaderAddr != "" {
			addr = reply.LeaderAddr
			continue
		}
		if reply.Error != "" {
			return errors.New(reply.Error)
		}
		return nil
	}
	return errors.New("too many redirects")
}

func (c *RPCClient) call(addr, method string, args interface{}, reply interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := rpcapi.Dial(ctx, addr)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Call(method, args, reply); err != nil {
		return fmt.Errorf("rpc call %s: %w", method, err)
	}
	return nil
}
