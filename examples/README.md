# NVIDIA Device API: Go Client Examples
This directory contains examples demonstrating how to use the NVIDIA Device API Go client across various integration patterns.

---

## Integration Patterns
| Example | Focus | Use Case |
| :--- | :--- | :--- |
| **[Basic Client](./client)** | **Point-in-Time Discovery** | CLI tools and scripts |
| **[Watch Monitor](./watch)** | **Stream Processing** | Real-time event-driven monitoring |
| **[Controller](./controller)** | **Standard Controllers** | `controller-runtime` / Kubebuilder |
| **[Fake Client](./fake-client)** | **Unit Testing** | Testing without a server |

---

## Usage
Most examples require a connection to a running Device API server.

1. **Start the mock server**: Follow the [setup instructions](../server/README.md).

2. **Configure the client**: Run the unique `export` command printed by the server in your current terminal:
```bash
export NVIDIA_DEVICE_API_SOCK=unix://<path_to_socket>
```

3. **Run an Example**: Navigate to an example directory or run it directly:
```bash
go run ./client/main.go
```

---
