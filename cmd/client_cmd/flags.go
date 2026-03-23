package client_cmd

import "github.com/spf13/cobra"

var serverAddr string

func addServerAddrFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&serverAddr, "addr", "localhost:15213", "breeoche server address")
}
