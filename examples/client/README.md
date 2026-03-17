# NVIDIA Device API: Basic Client Example
This example demonstrates how to perform point-in-time discovery of node-local resources using the NVIDIA Device API Go client.

---

## Usage

1. **Start the mock server**: Follow the [setup instructions](../server/README.md).

2. **Configure the client**: Run the unique `export` command printed by the server in your current terminal:
```bash
export NVIDIA_DEVICE_API_SOCK=unix://<path_to_socket>
```

3. **Run**:
```bash
go run main.go
```

> [!TIP]
> If the client fails to connect, ensure the `NVIDIA_DEVICE_API_SOCK` path matches the current server output.

---
