package registry

import (
	"sync"

	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/api"
)

var (
	lock      sync.RWMutex
	providers []api.ServiceProvider
)

func Register(p api.ServiceProvider) {
	lock.Lock()
	defer lock.Unlock()
	providers = append(providers, p)
}

func All() []api.ServiceProvider {
	lock.RLock()
	defer lock.RUnlock()
	cp := make([]api.ServiceProvider, len(providers))
	copy(cp, providers)
	return cp
}
