package api

import (
	"github.com/nvidia/nvsentinel/pkg/storage"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type RegistrationFunc func(f storage.StorageFactory, nodeName string, svr *grpc.Server) error

type APIGroupInfo struct {
	GroupName           string
	VersionedInstallers map[string]RegistrationFunc
}

type ServiceProvider interface {
	GroupName() string
	GroupVersion() schema.GroupVersion
	BuildGroupInfo() *APIGroupInfo
}
