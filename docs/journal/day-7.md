# Day 7 — client-go: Informers, Listers, and the Watch/List Pattern

## What is client-go?

`client-go` (`k8s.io/client-go`) is the official Go client library for talking to
a Kubernetes cluster. It provides:

| Package | Purpose |
|---------|---------|
| `kubernetes` | Typed clientset for every built-in API group (Pods, Deployments, …) |
| `dynamic` | Untyped client that works with arbitrary resources via `unstructured.Unstructured` |
| `discovery` | Discover which API groups/versions/resources the cluster supports |
| `tools/cache` | Informers, Listers, Indexers, work queues — the building blocks of every controller |
| `transport` | Low-level HTTP transport, auth, TLS |

For KubeSchedule the most critical package is **`tools/cache`** because it
contains the machinery that lets our controller react to changes in
`ScheduledAction` resources efficiently.

---

## The Watch/List Pattern

Kubernetes exposes two complementary operations on every resource:

1. **List** — returns the full set of objects that currently exist
   (`GET /apis/apps/v1/deployments`).
2. **Watch** — opens a long-lived HTTP connection that streams change events
   (ADDED, MODIFIED, DELETED) as they happen
   (`GET /apis/apps/v1/deployments?watch=true`).

A naive controller could poll with List on a timer, but that is wasteful and
adds latency. The **watch/list pattern** combines both:

```
1.  Initial List   →  get the full state of the world (+ a resourceVersion)
2.  Watch from RV  →  stream only the deltas from that point forward
3.  On error/410   →  re-list to resync, then watch again
```

This gives you **near-real-time** notifications with **minimal API server load**.
client-go wraps this entire cycle in a component called the **Reflector**.

---

## Core Components (tools/cache)

### Reflector

The Reflector is the lowest-level piece. It runs a `ListAndWatch` loop:

- Calls the **List** API to get all objects and their `resourceVersion`.
- Opens a **Watch** starting from that version.
- Pushes every event (add / update / delete) into a **DeltaFIFO** queue.
- If the watch expires or errors out, it re-lists and restarts the watch.

You rarely create a Reflector directly; Informers do it for you.

### Informer

An Informer sits on top of the Reflector. It:

1. **Pops objects from the DeltaFIFO** queue (via `processLoop`).
2. **Stores them in an in-memory cache** (the Indexer / Store) so you can read
   objects without hitting the API server.
3. **Dispatches events** to registered **ResourceEventHandlers** — the callbacks
   your controller implements:

```go
informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc:    func(obj interface{}) { enqueue(obj) },
    UpdateFunc: func(old, new interface{}) { enqueue(new) },
    DeleteFunc: func(obj interface{}) { enqueue(obj) },
})
```

These handlers should be lightweight: extract the object key
(`namespace/name`) and push it onto a **work queue**. The heavy work happens
later when the queue is drained.

### SharedInformer and SharedInformerFactory

Creating one Informer per controller per resource type is wasteful when
multiple controllers watch the same kind. A **SharedInformer** lets many
handlers share a single Reflector and cache. In practice you use a
**SharedInformerFactory** to get or create shared informers by GVR:

```go
factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
deployInformer := factory.Apps().V1().Deployments()

// Register handlers
deployInformer.Informer().AddEventHandler(...)

// Start all informers
factory.Start(stopCh)

// Wait for the caches to sync (initial List is complete)
factory.WaitForCacheSync(stopCh)
```

The `resyncPeriod` (30 s above) causes the Informer to periodically re-deliver
every cached object to your handlers as a synthetic Update event, so your
controller can self-heal even if it missed something.

### Lister

A **Lister** is a thin read-only wrapper around the Informer's local cache.
Instead of calling the API server, it reads directly from the in-memory
Indexer:

```go
lister := deployInformer.Lister()

// List all Deployments in a namespace (zero API calls)
deploys, err := lister.Deployments("default").List(labels.Everything())

// Get a single Deployment by name
deploy, err := lister.Deployments("default").Get("api-server")
```

**Why Listers matter:**

- They are **fast** — O(1) reads from an in-memory map.
- They are **consistent** — backed by the same watch stream as your event
  handlers, so the data is at most one watch-event behind.
- They **reduce API server load** — dozens of reconcile loops can read the
  cache without sending any network request.

### Indexer

The Indexer is the underlying store that supports custom secondary indices.
By default every object is indexed by `namespace/name`. You can add extra
indices (e.g., by label selector or by owner reference) for O(1) lookups:

```go
indexer.AddIndexers(cache.Indexers{
    "byOwner": func(obj interface{}) ([]string, error) {
        meta := obj.(metav1.Object)
        var keys []string
        for _, ref := range meta.GetOwnerReferences() {
            keys = append(keys, string(ref.UID))
        }
        return keys, nil
    },
})
```

---

## How It All Fits Together

```
API Server
    │
    │  List + Watch (HTTP)
    ▼
┌──────────┐
│ Reflector │──── ListAndWatch loop
└────┬─────┘
     │  push events
     ▼
┌───────────┐
│ DeltaFIFO │
└────┬──────┘
     │  pop events
     ▼
┌──────────┐     ┌────────────────────────────────┐
│ Informer │────►│ Indexer / Store (in-memory cache)│◄── Lister reads from here
└────┬─────┘     └────────────────────────────────┘
     │
     │  dispatch to handlers
     ▼
┌────────────────────────┐
│ ResourceEventHandlers  │
│  → enqueue object key  │
└────────┬───────────────┘
         │
         ▼
┌────────────────┐
│   Work Queue   │
└───────┬────────┘
        │
        ▼
┌───────────────────────────┐
│ Reconcile / processItem() │  ◄── YOUR controller logic
└───────────────────────────┘
```

---

## Key Takeaways for KubeSchedule

1. **We will never poll.** Our controller will use an Informer to watch
   `ScheduledAction` resources. The Reflector handles list/watch, reconnection,
   and re-sync automatically.

2. **Listers keep reconcile fast.** Inside `Reconcile()` we will use Listers to
   read Deployments, ConfigMaps, or any target resource without hitting the API
   server.

3. **SharedInformerFactory avoids duplicate watches.** If we ever need to watch
   multiple resource types (e.g., Deployments + StatefulSets), the factory
   ensures one watch per kind.

4. **The work queue decouples detection from processing.** Event handlers
   enqueue a key; the reconciler drains the queue at its own pace, with
   rate-limiting and retries built in.

5. **controller-runtime (used by Kubebuilder) wraps all of this.** When we
   scaffold our controller in Phase 3, `ctrl.NewManager` creates the shared
   informers, caches, and work queues for us — but understanding the primitives
   here means we can debug and tune them.

---

*Sources: [client-go README](https://github.com/kubernetes/client-go),
[client-go under the hood](https://github.com/kubernetes/sample-controller/blob/master/docs/controller-client-go.md),
[tools/cache GoDoc](https://pkg.go.dev/k8s.io/client-go/tools/cache)*
