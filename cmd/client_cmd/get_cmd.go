package client_cmd

import (
	"fmt"
	"github.com/Wregret/breeoche/client"
	"github.com/spf13/cobra"
)

var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "get the value by key",
	Long:  "get the value by key from breeoche server",
	Run:   doGet,
	Args:  cobra.ExactArgs(1),
}

func doGet(cmd *cobra.Command, args []string) {
	key := args[0]
	c := client.NewClient()
	value, err := c.Get(key)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(value)
}
