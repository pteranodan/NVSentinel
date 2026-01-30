package metrics

import (
	"sync"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

// ServerMetrics wraps the gRPC metrics and the private registry to avoid
// collisions with global metrics (e.g., Kine/etcd).
type ServerMetrics struct {
	Registry     *prometheus.Registry
	Collectors   *grpcprom.ServerMetrics
	registerOnce sync.Once
}

var (
	DefaultServerMetrics = &ServerMetrics{
		Registry: prometheus.NewRegistry(),
		Collectors: grpcprom.NewServerMetrics(
			grpcprom.WithServerHandlingTimeHistogram(),
		),
	}
)

func (m *ServerMetrics) Register() {
	m.registerOnce.Do(func() {
		if err := m.Registry.Register(m.Collectors); err != nil {
			klog.ErrorS(err, "failed to register gRPC metrics to private registry")
		}
	})
}

func (m *ServerMetrics) GetGatherer() prometheus.Gatherer {
	return m.Registry
}
