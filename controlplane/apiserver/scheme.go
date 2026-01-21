package apiserver

import (
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func init() {
	_ = devicev1alpha1.AddToScheme(scheme)
}
