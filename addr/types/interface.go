package types

type Interface interface {
	Fetch() (*Addr, error)
}

type Creator func() Interface

var Plugins = map[string]Creator{}