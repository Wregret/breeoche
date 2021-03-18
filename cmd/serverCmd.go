package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var port int

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start breeoche server",
	Long:  "start breeoche server to receive operation on storage",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(os.Stdout, "breeoche server start on port: %d\n", port)
	},
}

func init() {
	serverCmd.Flags().IntVarP(&port, "port", "p", 15213, "specify the port number of brioiche server")
}
