# TODO(pteranodan)

---

### General

- [ ] **complete OSRB**

- [ ] standardize project layout
- [ ] trim down / consolidate docs
- [ ] finalize default target / socket path (`/run/nvidia/device-api/api.sock`)
- [ ] implement discovery api
- [ ] introduce internal superset type
- [ ] ?generate protos from Go types? (SSOT)
- [ ] add validation to protos

---

### Server

- [ ] **design doc**

- [ ] auth interceptor
- [ ] validation interceptor
- [ ] audit logs
- [x] health checks
- [x] metrics server
- [x] admin interface
- [ ] versioning
  - [ ] version metric
  - [ ] version endpoint
  - [ ] ?add version to response header?
- [ ] datastore
  - [ ] Kine compaction control
  - [ ] SQLite compaction control
  - [ ] ?export db size metric?
  - [ ] ?server-side cache?
- [ ] server-gen
  - [ ] fake server
    - [ ] use generated fake server in examples
- [ ] deployment
  - [ ] image
  - [ ] helm chart
  - [ ] docs
- [ ] unit tests
- [ ] integration tests
- [ ] performance tests

---

### Client

- [ ] ?add version to request header?

- [ ] client-gen
  - [ ] ?discovery api?
    - removes the need for users to provide a RESTMapper to controller-runtime manager
  - [ ] implement updateStatus template
  - [ ] implement patch template
  - [ ] ?aggregated clientset w/ standard k8s clientset?
  - [ ] integration tests

---
