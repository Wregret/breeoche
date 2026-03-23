package server_cmd

import (
	"errors"
	"fmt"
	"github.com/Wregret/breeoche/server"
	"github.com/spf13/cobra"
	"strings"
)

var (
	port              int
	host              string
	nodeID            string
	peers             string
	dataDir           string
	snapshotThreshold int
	debug             bool
)

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "start breeoche server",
	Long:  "start breeoche server to receive operation on storage",
	RunE: func(cmd *cobra.Command, args []string) error {
		if nodeID == "" {
			return errors.New("--id is required")
		}
		addr := fmt.Sprintf("%s:%d", host, port)
		peerMap, err := parsePeers(peers)
		if err != nil {
			return err
		}

		s, err := server.NewServer(server.Config{
			ID:                nodeID,
			Addr:              addr,
			Peers:             peerMap,
			DataDir:           dataDir,
			SnapshotThreshold: snapshotThreshold,
			Debug:             debug,
		})
		if err != nil {
			return err
		}
		return s.Start()
	},
}

func init() {
	ServerCmd.Flags().IntVarP(&port, "port", "p", 15213, "specify the port number of breeoche server")
	ServerCmd.Flags().StringVar(&host, "host", "127.0.0.1", "host interface to bind")
	ServerCmd.Flags().StringVar(&nodeID, "id", "", "unique node id")
	ServerCmd.Flags().StringVar(&peers, "peers", "", "comma-separated list of id=host:port")
	ServerCmd.Flags().StringVar(&dataDir, "data-dir", "data", "directory for raft state")
	ServerCmd.Flags().IntVar(&snapshotThreshold, "snapshot-threshold", 100, "entries between automatic snapshots")
	ServerCmd.Flags().BoolVar(&debug, "debug", false, "enable verbose debug logging")
}

func parsePeers(raw string) (map[string]string, error) {
	result := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return result, nil
	}
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid peer entry: %s", pair)
		}
		id := strings.TrimSpace(parts[0])
		addr := strings.TrimSpace(parts[1])
		if id == "" || addr == "" {
			return nil, fmt.Errorf("invalid peer entry: %s", pair)
		}
		result[id] = addr
	}
	return result, nil
}
