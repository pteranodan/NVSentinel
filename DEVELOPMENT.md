# TODO

```bash 
$ sudo mkdir -p /var/run/nvidia-device-apiserver
$ sudo chmod 755 /var/lib/nvidia-device-apiserver 
```

```bash
$ sudo ./bin/device-apiserver --version
$ sudo ./bin/device-apiserver --version=raw

$ sudo ./bin/device-apiserver --help

$ sudo ./bin/device-apiserver --v=2 2>&1 | grep -v "etcd-client"

## Admin: Channelz, Reflection, and Health
$ sudo grpcurl -plaintext localhost:50051 list

$ sudo grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

## CREATE
$ sudo grpcurl -plaintext \
  -unix -d '{
    "gpu": {
      "metadata": {
        "name": "gpu-7",
        "namespace": "default"
      },
      "spec": {
        "uuid": "GPU-a1b2c3d4-e5f6-a7b8-c9d0-e1f2a3b4c5d6"
      }
    }
  }' \
  unix:/var/run/nvidia-device-api/device-api.sock \
  nvidia.nvsentinel.v1alpha1.GpuService/CreateGpu

## GET
$ sudo grpcurl -plaintext \
  -unix -d '{"name": "gpu-7", "namespace": "default"}' \
  unix:/var/run/nvidia-device-api/device-api.sock \
  nvidia.nvsentinel.v1alpha1.GpuService/GetGpu

## UPDATE
$ sudo grpcurl -plaintext \
  -unix -d '{
    "gpu": {
      "metadata": {
        "name": "gpu-7",
        "namespace": "default",
        "resource_version": "<rv>"
      },
      "spec": {
        "uuid": "GPU-a1b2c3d4-e5f6-a7b8-c9d0-e1f2a3b4c5d6"
      },
      "status": {
        "conditions": [
          {
            "type": "Ready",
            "status": "True",
            "reason": "DriverReady",
            "message": "Driver is posting a Ready status."
          }
        ]
      }
    }
  }' \
  unix:/var/run/nvidia-device-api/device-api.sock \
  nvidia.nvsentinel.v1alpha1.GpuService/UpdateGpu

## GET
$ sudo grpcurl -plaintext \
  -unix -d '{"name": "gpu-7", "namespace": "default"}' \
  unix:/var/run/nvidia-device-api/device-api.sock \
  nvidia.nvsentinel.v1alpha1.GpuService/GetGpu

## LIST
$ sudo grpcurl -plaintext \
  -unix -d '{"namespace": "", "opts": {"resource_version": "0"}}' \
  unix:/var/run/nvidia-device-api/device-api.sock \
  nvidia.nvsentinel.v1alpha1.GpuService/ListGpus
  ```
