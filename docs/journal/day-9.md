# Day 9 — Using client-go: Kubeconfig, Clientset, and Listing Resources

## Overview

Today we wrote our first program that talks to a real Kubernetes cluster using
`client-go`. This journal documents the key concepts behind `cmd/explore/main.go`.

---

## Kubeconfig and REST Config

Every client-go program starts by building a `*rest.Config` — the connection
settings (server URL, auth credentials, TLS) needed to talk to the API server.

### Where the kubeconfig lives

| Source | How to use |
|--------|-----------|
| `~/.kube/config` (default) | `clientcmd.RecommendedHomeFile` resolves the absolute path |
| `KUBECONFIG` env var | `clientcmd.NewDefaultClientConfigLoadingRules()` checks this automatically |
| `--kubeconfig` flag | Pass the path as the second arg to `BuildConfigFromFlags` |
| In-cluster (Pod) | `rest.InClusterConfig()` reads the service account token mounted at `/var/run/secrets/` |

### Why `~` doesn't work in Go

The tilde (`~`) is a **shell expansion** feature. Go's `os.Open` and the rest of
the standard library treat it as a literal character. Always use
`clientcmd.RecommendedHomeFile` or `os.UserHomeDir()` + `filepath.Join()`
instead.

### BuildConfigFromFlags

```go
config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
```

- First argument: API server URL override (empty = read from kubeconfig).
- Second argument: path to the kubeconfig file.
- Returns a `*rest.Config` with host, TLS, bearer token, etc.

---

## The Clientset

```go
clientset, err := kubernetes.NewForConfig(config)
```

A **Clientset** is a collection of typed clients — one per API group/version.
It gives you strongly-typed methods for every built-in Kubernetes resource:

```
clientset.CoreV1().Pods(ns)            // PodInterface
clientset.AppsV1().Deployments(ns)     // DeploymentInterface
clientset.BatchV1().Jobs(ns)           // JobInterface
clientset.CoreV1().ConfigMaps(ns)      // ConfigMapInterface
```

Each of these returns an interface with `Get`, `List`, `Create`, `Update`,
`Patch`, `Delete`, and `Watch` methods.

---

## kubernetes.Interface vs *kubernetes.Clientset

This is one of the most important design decisions in the code:

```go
func listDeployments(ctx context.Context, client kubernetes.Interface, namespace string) ([]string, error)
```

| Type | What it is | When to use |
|------|-----------|-------------|
| `*kubernetes.Clientset` | Concrete struct | Only in `main()` when creating it |
| `kubernetes.Interface` | Interface satisfied by both the real Clientset and `fake.NewSimpleClientset` | Everywhere else — function params, struct fields |

By accepting `kubernetes.Interface`, our function works with:

- A **real clientset** connected to a cluster (production / `main()`).
- A **fake clientset** pre-loaded with test objects (unit tests).

This is the **dependency inversion principle** in action: depend on
abstractions, not concretions. It's the same reason we defined `Executor`,
`Validator`, and `StatusReporter` as interfaces back in Day 2.

---

## Listing Deployments

```go
deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
```

### Breaking it down

1. `client.AppsV1()` — selects the `apps/v1` API group client.
2. `.Deployments(namespace)` — scopes to Deployments in the given namespace.
   Pass `""` (empty string) to list across all namespaces.
3. `.List(ctx, metav1.ListOptions{})` — performs `GET /apis/apps/v1/namespaces/{ns}/deployments`.

### metav1.ListOptions

An empty `ListOptions{}` means "return everything." You can filter with:

| Field | Example | Effect |
|-------|---------|--------|
| `LabelSelector` | `"app=nginx"` | Only objects matching the label |
| `FieldSelector` | `"metadata.name=api"` | Only objects matching the field (limited support) |
| `Limit` | `100` | Paginate — return at most 100 items |
| `Continue` | (token from previous response) | Fetch the next page |

### The result

`List` returns a `*appsv1.DeploymentList` with an `.Items` field — a
`[]appsv1.Deployment` slice. Each item has the full Deployment spec and status,
including `ObjectMeta` (name, namespace, labels, annotations, etc.).

---

## Context

```go
names, err := listDeployments(context.Background(), clientset, "default")
```

Every client-go API call takes a `context.Context` as its first argument. This
allows:

- **Cancellation** — cancel a long-running List if the caller shuts down.
- **Timeouts** — `context.WithTimeout(ctx, 5*time.Second)` prevents hanging.
- **Value propagation** — pass request-scoped data (logger, trace ID) down the
  call chain.

For a short-lived CLI tool, `context.Background()` is fine. In a controller's
`Reconcile` method, you'll receive a context that's cancelled when the manager
shuts down.

---

## The Fake Clientset (preview for Day 10)

`k8s.io/client-go/kubernetes/fake` provides `NewSimpleClientset`:

```go
fakeClient := fake.NewSimpleClientset(objects...)
```

It creates an **in-memory Kubernetes API** that implements `kubernetes.Interface`.
You seed it with runtime objects, and your code can't tell the difference from a
real cluster. This is why accepting `kubernetes.Interface` matters — it makes the
swap seamless.

---

## Key Takeaways

1. **Always use `clientcmd.RecommendedHomeFile`** — never hardcode `~/.kube/config`.
2. **`kubernetes.Interface` is the abstraction** — accept it in functions and
   struct fields so you can test with fakes.
3. **The clientset is organised by API group** — `CoreV1()`, `AppsV1()`,
   `BatchV1()`, etc.
4. **Context flows through every call** — it enables cancellation, timeouts, and
   value propagation.
5. **`metav1.ListOptions`** controls filtering and pagination.
6. **Extracting logic out of `main()`** into functions that accept interfaces is
   the foundation of testable Kubernetes code.

---

*References:
[client-go godoc](https://pkg.go.dev/k8s.io/client-go),
[clientcmd package](https://pkg.go.dev/k8s.io/client-go/tools/clientcmd),
[fake package](https://pkg.go.dev/k8s.io/client-go/kubernetes/fake)*
