# NVIDIA Device API Helm Chart

This Helm chart deploys the NVIDIA `device-apiserver` as a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) on a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.25+
- [Helm](https://helm.sh/) 3.16.1+

## Installing the Chart

```bash
helm repo add <YOUR_REPO_NAME> <YOUR_REPO_URL>
helm install my-device-apiserver <YOUR_REPO_NAME>/device-apiserver
```

### Customizing the Deployment
The server defaults to an in-memory database for maximum performance. You can use `extraArgs` to: enable persistence, tune performance & responsiveness, manage the server's lifecycle, or increase logging.

A complete list of available flags can be found by running `device-apiserver --help`.

```bash
# Example: Enabling disk persistence, verbose logging, 10s watch heartbeats, and a 60s graceful shutdown
helm install my-device-apiserver <YOUR_REPO_NAME>/device-apiserver \
  --set "extraArgs={--database-path=/var/lib/nvidia-device-apiserver/state.db,-v=4,--etcd-watch-progress-notify-interval=10s,--shutdown-grace-period=60s}"
```
**Note**: If enabling persistence, ensure that the directory (e.g., `/var/lib/nvidia-device-apiserver/`) is defined in `extraVolumes` and `extraVolumeMounts` to persist data across pod restarts.

## Uninstalling the Chart

```bash
helm uninstall my-device-apiserver
```

## Configuration

For a full list of chart-specific parameters, see the [values.yaml](values.yaml) file. To see all available binary flags, run the container locally with the `--help` flag.
