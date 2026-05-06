# kubeschedule
Schedule any K8s operation

## Prerequisites

- Go 1.26+
- [controller-gen](https://github.com/kubernetes-sigs/controller-tools) for code generation

```bash
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
```

## Usage

```bash
make all        # lint + test + build
make generate   # generate DeepCopy methods and CRD manifests
make build      # build binaries
make test       # run tests
make lint       # run golangci-lint
make clean      # remove build artifacts
```

## Code Generation

The API types in `api/v1alpha1/types.go` use kubebuilder markers to drive code generation.

Running `make generate` will:

1. **Generate DeepCopy methods** — creates `api/v1alpha1/zz_generated.deepcopy.go` with `DeepCopyInto()`, `DeepCopy()`, and `DeepCopyObject()` implementations.
2. **Generate CRD manifests** — creates the `ScheduledAction` CRD YAML in `config/crd/`.

You can also run `controller-gen` directly:

```bash
# DeepCopy only
controller-gen object paths=./api/v1alpha1/...

# CRD manifests only
controller-gen crd paths=./api/v1alpha1/... output:crd:dir=./config/crd

# Both at once
controller-gen object crd paths=./api/v1alpha1/... output:crd:dir=./config/crd
```

### controller-gen arguments reference

| Argument | Description |
|----------|-------------|
| `object` | Generate `DeepCopy` / `DeepCopyObject` methods |
| `crd` | Generate CRD YAML manifests from kubebuilder markers |
| `paths=./api/v1alpha1/...` | Go packages to scan (recursive) |
| `output:crd:dir=./config/crd` | Output directory for generated CRDs |
