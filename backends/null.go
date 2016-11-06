package backends

import guerrilla "github.com/flashmob/go-guerrilla"

func init() {
	backends["null"] = &NullBackend{}
}

type NullBackend struct{}

func (n *NullBackend) Initialize(backendConfig guerrilla.BackendConfig) error {}
