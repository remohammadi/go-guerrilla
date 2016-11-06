package backends

import (
	guerrilla "github.com/flashmob/go-guerrilla"
	"fmt"
)

var backends = map[string]guerrilla.Backend

func New(backendName string, backendConfig guerrilla.BackendConfig) (guerrilla.Backend, error) {
  backend, found := backends[backendName]
  if ! found {
    return nil, fmt.Errorf("backend %q not found", backendName)
  }
  err := backend.Initialize(backendConfig)
  if err != nil {
    return nil, fmt.Errorf("error while initializing the backend: %s", err)
  }
  return backend, nil
}
