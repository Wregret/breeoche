package cmd

import (
	"fmt"
	"github.com/Wregret/breeoche/cmd/client_cmd"
	"github.com/Wregret/breeoche/cmd/server_cmd"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "breeoche",
	Short: "breeoche is a simple key/value storage service",
	Long:  `breeoche is a simple key/value storage service for study purpose`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(server_cmd.ServerCmd)

	rootCmd.AddCommand(client_cmd.PingCmd)

	rootCmd.AddCommand(client_cmd.GetCmd)
	rootCmd.AddCommand(client_cmd.SetCmd)
	rootCmd.AddCommand(client_cmd.InsertCmd)
	rootCmd.AddCommand(client_cmd.DeleteCmd)
}
