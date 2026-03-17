# NVIDIA Device API: Controller Example
This example demonstrates how to integrate the NVIDIA Device API with **controller-runtime** and **Kubebuilder**-generated controllers

---

## Usage

1. **Start the mock server**: Follow the [setup instructions](../server/README.md).

2. **Configure the client**: Run the unique `export` command printed by the server in your current terminal:
```bash
export NVIDIA_DEVICE_API=unix://<path_to_socket>
```

3. **Run**:
```bash
go run main.go
```

> [!TIP]
> If the client fails to connect, ensure the `NVIDIA_DEVICE_API` path matches the current server output.

---
