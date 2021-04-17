package client_cmd

import (
	"fmt"
	"github.com/Wregret/breeoche/client"
	"github.com/spf13/cobra"
)

var InsertCmd = &cobra.Command{
	Use:   "insert",
	Short: "insert a key-value pair",
	Long:  "insert a key-value pair to breeoche server. fail if key exists",
	Run:   doInsert,
	Args:  cobra.ExactArgs(2),
}

func doInsert(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]
	c := client.NewClient()
	err := c.Insert(key, value)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("succeed!")
}
