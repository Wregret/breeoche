package rpcapi

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
)

// Dial establishes an RPC client using the standard RPC HTTP CONNECT handshake.
func Dial(ctx context.Context, addr string) (*rpc.Client, error) {
	return dialHTTPPath(ctx, "tcp", addr, RPCPath)
}

func dialHTTPPath(ctx context.Context, network, addr, path string) (*rpc.Client, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(conn, "CONNECT %s HTTP/1.0\r\n\r\n", path)
	if err != nil {
		conn.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, errors.New("rpc: connect failed: " + resp.Status)
	}

	return rpc.NewClient(conn), nil
}
