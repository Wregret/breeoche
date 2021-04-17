package client_cmd

import (
	"fmt"
	"github.com/Wregret/breeoche/client"
	"github.com/spf13/cobra"
)

var PingCmd = &cobra.Command{
	Use:   "ping",
	Short: "ping breeoche server",
	Long:  "ping breeoche server. get pong! if server is up and running",
	Run:   doPing,
}

func doPing(cmd *cobra.Command, args []string) {
	c := client.NewClient()
	value, err := c.Ping()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(value)
}
