package client_cmd

import (
	"fmt"
	"github.com/Wregret/breeoche/client"
	"github.com/spf13/cobra"
)

var SetCmd = &cobra.Command{
	Use:   "set",
	Short: "set a key-value pair",
	Long:  "set a key-value pair to breeoche server. overwrite the value if key exists",
	Run:   doSet,
	Args:  cobra.ExactArgs(2),
}

func doSet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]
	c := client.NewClient()
	err := c.Set(key, value)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("succeed!")
}
