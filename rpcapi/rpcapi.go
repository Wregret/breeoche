package rpcapi

const (
	RPCPath          = "/rpc"
	KVServiceName    = "KVService"
	RaftServiceName  = "RaftService"
	ErrNotLeader     = "not leader"
	ErrLeaderUnknown = "leader unknown"
)

type PingArgs struct{}

type PingReply struct {
	Message string
	Error   string
}

type GetArgs struct {
	Key string
}

type GetReply struct {
	Value      string
	Found      bool
	Error      string
	LeaderAddr string
}

type SetArgs struct {
	Key   string
	Value string
}

type InsertArgs struct {
	Key   string
	Value string
}

type DeleteArgs struct {
	Key string
}

type MutateReply struct {
	Error      string
	LeaderAddr string
}

type HealthArgs struct{}

type HealthStatus struct {
	ID           string `json:"id"`
	Term         int    `json:"term"`
	State        string `json:"state"`
	LeaderID     string `json:"leader_id"`
	CommitIndex  int    `json:"commit_index"`
	LastApplied  int    `json:"last_applied"`
	LastLogIndex int    `json:"last_log_index"`
}

type HealthReply struct {
	Status HealthStatus
	Error  string
}
