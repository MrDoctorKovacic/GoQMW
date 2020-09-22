package module

// Module specifies the start and stop for MDroid modules
type Module interface {
	Start() error
	Stop() error
}
