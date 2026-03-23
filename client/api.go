package client

// API defines the client operations supported by Breeoche.
type API interface {
	Ping() (string, error)
	Get(key string) (string, error)
	Set(key string, value string) error
	Insert(key string, value string) error
	Delete(key string) error
}
