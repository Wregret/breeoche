package client_cmd

import (
	"fmt"
	"github.com/Wregret/breeoche/client"
	"github.com/spf13/cobra"
)

var DeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete the key-value pair by key",
	Long:  "delete the key-value pair by key from breeoche server. no-op if key doesn't exists",
	Run:   doDelete,
	Args:  cobra.ExactArgs(1),
}

func doDelete(cmd *cobra.Command, args []string) {
	key := args[0]
	c := client.NewClient()
	err := c.Delete(key)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("succeed!")
}
