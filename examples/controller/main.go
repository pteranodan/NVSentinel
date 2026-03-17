//  Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

// TODO: fix broken example

// Package main demonstrates how to use the NVIDIA Device API with a standard controller.
package main

import (
	"context"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/pkg/client-go/clientset/versioned"
	"github.com/nvidia/nvsentinel/pkg/grpc/client"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	// Register NVIDIA Device types
	utilruntime.Must(devicev1alpha1.AddToScheme(scheme))
	// Register Standard K8s types for hybrid use
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	setupLog := ctrl.Log.WithName("setup")

	k8sConfig := ctrl.GetConfigOrDie()

	// Resolve the connection target from the environment or use the default socket path.
	target := os.Getenv(client.DeviceAPISocketEnvVar)
	if target == "" {
		target = client.DefaultDeviceAPISocketPath
	}

	/*
		// Initialize the device Clientset.
		deviceConfig := &client.Config{Target: target}
		deviceClientset := versioned.NewForConfigOrDie(context.TODO(), deviceConfig)

		factory := informers.NewSharedInformerFactory(deviceClientset, 10*time.Minute)
		gpuInformer := factory.Device().V1alpha1().GPUs().Informer()

		// Initialize the controller-runtime Manager.
		// A dummy rest.Config is used here as we are not connecting to a real K8s API server.
		mgr, err := ctrl.NewManager(k8sConfig, ctrl.Options{
			Scheme: scheme,
			MapperProvider: func(c *rest.Config, httpClient *http.Client) (meta.RESTMapper, error) {
				mapper, err := apiutil.NewDynamicRESTMapper(c, httpClient)
				if err != nil {
					return nil, err
				}
				fixedMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{devicev1alpha1.SchemeGroupVersion})
				fixedMapper.Add(devicev1alpha1.SchemeGroupVersion.WithKind("GPU"), meta.RESTScopeRoot)
				return meta.FirstHitRESTMapper{MultiRESTMapper: []meta.RESTMapper{mapper, fixedMapper}}, nil
			},
			NewClient: func(config *rest.Config, opts ctrlclient.Options) (ctrlclient.Client, error) {
				return deviceclient.New(k8sConfig, deviceConfig, opts)
			},
			Cache: cache.Options{
				NewInformer: func(lw toolscache.ListerWatcher, obj runtime.Object, resync time.Duration, indexers toolscache.Indexers) toolscache.SharedIndexInformer {
					if _, ok := obj.(*devicev1alpha1.GPU); ok {
						if err := gpuInformer.AddIndexers(indexers); err != nil {
							setupLog.V(1).Info("Informer already has some of these indexers", "error", err)
						}
						return gpuInformer
					}
					return toolscache.NewSharedIndexInformer(lw, obj, resync, indexers)
				},
			},
		})
		if err != nil {
			setupLog.Error(err, "unable to create manager")
			os.Exit(1)
		}

		if err = (&GPUReconciler{
			Client: mgr.GetClient(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "gpu")
			os.Exit(1)
		}

		ctx := ctrl.SetupSignalHandler()

		setupLog.Info("starting manager")

		if err := mgr.Start(ctx); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	*/
}

// GPUReconciler reconciles GPUs using the local gRPC cache.
type GPUReconciler struct {
	ctrlclient.Client
}

func (r *GPUReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	list := &devicev1alpha1.GPUList{}
	if err := r.List(ctx, list); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Cache Status", "total_gpus_found", len(list.Items))
	for _, g := range list.Items {
		log.Info("GPU in Cache", "name", g.Name, "ns", g.Namespace)
	}

	var gpu devicev1alpha1.GPU
	// The Get call is transparently routed through the injected gRPC-backed informer.
	if err := r.Get(ctx, req.NamespacedName, &gpu); err != nil {
		return ctrl.Result{}, ctrlclient.IgnoreNotFound(err)
	}

	log.Info("Reconciled GPU", "name", gpu.Name, "uuid", gpu.Spec.UUID)

	return ctrl.Result{}, nil
}

func (r *GPUReconciler) SetupWithManager(mgr ctrl.Manager) error {
	skipNameValidation := true
	return ctrl.NewControllerManagedBy(mgr).
		For(&devicev1alpha1.GPU{}).
		WithOptions(controller.TypedOptions[reconcile.Request]{
			SkipNameValidation: &skipNameValidation,
		}).
		Complete(r)
}

// --- Subresource Operations ---

func (c *DynamicClient) Status() ctrlclient.SubResourceWriter {
	return &dynamicStatusWriter{
		SubResourceWriter: c.Client.Status(),
		deviceClient:      c.DeviceClient,
	}
}

type dynamicStatusWriter struct {
	ctrlclient.SubResourceWriter
	deviceClient versioned.Interface
}

func (w *dynamicStatusWriter) Update(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.SubResourceUpdateOption) error {
	if gpu, ok := obj.(*devicev1alpha1.GPU); ok {
		_, err := w.deviceClient.DeviceV1alpha1().GPUs().UpdateStatus(ctx, gpu, metav1.UpdateOptions{})
		return err
	}
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}

func (w *dynamicStatusWriter) Patch(ctx context.Context, obj ctrlclient.Object, patch ctrlclient.Patch, opts ...ctrlclient.SubResourcePatchOption) error {
	if gpu, ok := obj.(*devicev1alpha1.GPU); ok {
		data, err := patch.Data(gpu)
		if err != nil {
			return err
		}
		_, err = w.deviceClient.DeviceV1alpha1().GPUs().Patch(ctx, gpu.Name, types.PatchType(patch.Type()), data, metav1.PatchOptions{}, "status")
		return err
	}
	return w.SubResourceWriter.Patch(ctx, obj, patch, opts...)
}
