package client_cmd

import (
	"github.com/Wregret/breeoche/client"
	"github.com/spf13/cobra"
)

var serverAddr string
var useHTTP bool

func addServerAddrFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&serverAddr, "addr", "localhost:15213", "breeoche server address")
	cmd.Flags().BoolVar(&useHTTP, "http", false, "use HTTP instead of RPC")
}

func newClient() client.API {
	if useHTTP {
		return client.NewHTTPClient(serverAddr)
	}
	return client.NewRPCClient(serverAddr)
}
