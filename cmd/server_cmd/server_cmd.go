package server_cmd

import (
	"github.com/Wregret/breeoche/server"
	"github.com/spf13/cobra"
	"strconv"
)

var port int

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "start breeoche server",
	Long:  "start breeoche server to receive operation on storage",
	Run: func(cmd *cobra.Command, args []string) {
		s := server.NewServer()
		s.Start(strconv.Itoa(port))
	},
}

func init() {
	ServerCmd.Flags().IntVarP(&port, "port", "p", 15213, "specify the port number of breeoche server")
}
