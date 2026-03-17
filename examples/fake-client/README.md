# NVIDIA Device API: Fake Client Example
This example demonstrates how to use the generated **fake versioned client** and **SharedInformerFactory** to test controller behavior without a running server.

---

## Usage
This example is executed as a standard Go test. It uses an in-memory `ObjectTracker` to simulate gRPC server states and resource events.

**Run the test**:
```bash
go test -v -run TestGPUInformerWithFakeClient
```

---
