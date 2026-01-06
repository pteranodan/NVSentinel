# NVIDIA Device API Go Client: Development Guide
This document outlines the workflow for developing and maintaining the `nvidia/client-go` SDK. Because this library provides Kubernetes-native interfaces, it relies heavily on **code generation**. Most of the code in this directory should not be edited manually.

## Prerequisites
* **Go**: Must match the version specified in `go.mod`.
* **Make**: Standard build tool.

## Structure
- `client/`: [Generated] The Clientset. **Do not edit manually.**
- `listers/`: [Generated] Type-safe listers. **Do not edit manually.**
- `informers/`: [Generated] Shared Index Informers. **Do not edit manually.**
- `nvgrpc/`: [Manual] The gRPC transport layer, interceptors, and connection management logic.
- `version/`: [Manual] Version injection functionality via `ldflags`.

## Workflow

### 1. Code Generation
To (re)generate the client, run:

```bash
# Downloads codegen tools and generates clients/listers/informers
make code-gen
```

> [!TIP]
> **Did you modify the API?**
>
> If you have changed the types in the `../api` module (Proto or Go), you must run `make code-gen` **inside that directory first**.
> This ensures that the low-level bindings (Protobufs, DeepCopy, Conversions) are up-to-date before this client attempts to generate the high-level interfaces.

### 2. Building & Testing
Verify that the generated code compiles and passes unit tests.

```bash
# Compile everything (verifies type safety of generated code)
make build

# Run unit tests (focuses on the transport layer and manual logic)
make test
```

### 3. Full Cycle
To run the complete pipeline (Generation → Test → Build) in one go:

```bash
make all
```

## Code Generation Pipeline
This SDK is automatically generated from the Protocol Buffer definitions and Go types found in the `../api` module. We use standard Kubernetes code generators, including a customized build of `client-gen` to handle gRPC transport natively.

```mermaid
graph TD
    API["API Definitions<br/>(nvidia/nvsentinel/api)"] -->|Input| CG(client-gen<br/>*Custom Build*)
    API -->|Input| LG(lister-gen)

    CG -->|Generates| CLIENT[client/versioned]
    LG -->|Generates| LISTERS[listers/]

    CLIENT & LISTERS -->|Input| IG(informer-gen)
    IG -->|Generates| INFORMERS[informers/]

    CLIENT & LISTERS & INFORMERS -->|Final Output| SDK[Ready-to-use SDK]
```

### Components
1. `client-gen` (**Custom**): Generates the **Clientset**. This provides access to the API Resources (e.g., `DeviceV1alpha1().GPUs()`) and maps standard Kubernetes verbs (Get, List, Watch) to the node-local gRPC transport.
2. `lister-gen`: Generates **Listers**. These provide a read-only, cached view of resources, allowing for fast lookups without making network calls.
3. `informer-gen`: Generates **Informers**. These coordinate the Client and Listers to watch for updates and sync the local cache.

### Modifying Generated Code
> [!WARNING]
> **Do not edit generated files directly.**
>
> Files in `client/`, `listers/`, and `informers/` are overwritten every time you run `make code-gen`.
> To change the client behavior, you must modify the generator source code in `../code-generator/cmd/client-gen` or the API definitions in the `../api` module.


## Housekeeping
If you need to reset your environment:

```bash
# Removes generated code (client, listers, informers)
make clean
```
