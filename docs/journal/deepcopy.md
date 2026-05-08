# DeepCopy in KubeSchedule — A Practical Guide

## Why DeepCopy Exists

Kubernetes controllers constantly pass objects between goroutines — the informer
cache, the reconciler, status updates, webhook handlers.  If two goroutines hold
a pointer to the **same** struct and one mutates it, the other sees corrupted
data (or panics).  DeepCopy creates a completely independent clone: every
pointer, slice, and map is duplicated so the copy can be mutated freely without
touching the original.

The rule is simple: **never mutate an object you got from the cache.  Always
work on a DeepCopy.**

---

## The Three Generated Methods

controller-gen produces up to **three** methods per struct.  Not every struct
gets all three.

### 1. `DeepCopyInto(out *T)`

Copies every field from `in` into an already-allocated `out`.  This is the
**workhorse** — all the real cloning logic lives here.

```go
// from zz_generated.deepcopy.go — ScheduledActionSpec
func (in *ScheduledActionSpec) DeepCopyInto(out *ScheduledActionSpec) {
    *out = *in                          // shallow copy all value-type fields
    if in.TimeZone != nil {             // pointer → allocate a new string
        in, out := &in.TimeZone, &out.TimeZone
        *out = new(string)
        **out = **in
    }
    in.Action.DeepCopyInto(&out.Action) // nested struct → recurse
    if in.Suspend != nil {              // pointer → allocate a new bool
        in, out := &in.Suspend, &out.Suspend
        *out = new(bool)
        **out = **in
    }
}
```

### 2. `DeepCopy() *T`

Convenience wrapper — allocates a new `T`, calls `DeepCopyInto`, returns it.

```go
func (in *ScheduledActionSpec) DeepCopy() *ScheduledActionSpec {
    if in == nil {
        return nil
    }
    out := new(ScheduledActionSpec)
    in.DeepCopyInto(out)
    return out
}
```

### 3. `DeepCopyObject() runtime.Object`

**Only generated for root objects** — types marked with
`+kubebuilder:object:root=true`.  This is the method that satisfies the
`runtime.Object` interface, which Kubernetes needs to store the object in its
internal registries and caches.

In our project, only `ScheduledAction` and `ScheduledActionList` get this:

```go
func (in *ScheduledAction) DeepCopyObject() runtime.Object {
    if c := in.DeepCopy(); c != nil {
        return c
    }
    return nil
}
```

---

## Which Structs Get Which Methods

| Struct                 | `DeepCopyInto` | `DeepCopy` | `DeepCopyObject` | Why                                                  |
|------------------------|:--------------:|:----------:|:-----------------:|------------------------------------------------------|
| `ScheduledAction`      | yes            | yes        | yes               | Root object (`+kubebuilder:object:root=true`)        |
| `ScheduledActionList`  | yes            | yes        | yes               | Root object (`+kubebuilder:object:root=true`)        |
| `ScheduledActionSpec`  | yes            | yes        | no                | Sub-struct with pointers (`*string`, `*bool`)        |
| `ScheduledActionStatus`| yes            | yes        | no                | Sub-struct with pointers and slices                  |
| `ActionSpec`           | yes            | yes        | no                | Sub-struct with a map (`map[string]string`)          |
| `TargetRef`            | yes            | yes        | no                | Sub-struct with a pointer (`*string`)                |
| `ExecutionHistory`     | yes            | yes        | no                | Sub-struct with a pointer (`*metav1.Duration`)       |
| `ActionType`           | none           | none       | no                | Pure value type (`type ActionType string`) — no copy needed |

---

## How controller-gen Decides What to Clone

controller-gen walks every field of each struct and picks a strategy based on
the Go type:

| Field type              | Strategy                                 | Example in our project                                    |
|-------------------------|------------------------------------------|-----------------------------------------------------------|
| Value type (`string`, `bool`, `int`) | `*out = *in` (shallow copy is enough) | `ActionType`, `Result`, `Message`                |
| Pointer (`*string`, `*bool`)         | `*out = new(T); **out = **in`         | `TimeZone *string`, `Suspend *bool`              |
| Pointer to struct (`*metav1.Time`)   | `*out = (*in).DeepCopy()`             | `LastRun *metav1.Time`                           |
| Pointer to struct w/o own DeepCopy   | `*out = new(T); **out = **in`         | `Duration *metav1.Duration`                      |
| Map (`map[K]V`)                      | Allocate new map, copy entries          | `Params map[string]string`                       |
| Slice of structs (`[]T`)             | `make([]T, len)` + `DeepCopyInto` loop  | `History []ExecutionHistory`, `Items []ScheduledAction` |
| Embedded struct with own DeepCopy    | `in.Field.DeepCopyInto(&out.Field)`   | `ObjectMeta`, `ListMeta`                         |
| Embedded struct (value-only fields)  | `*out = *in` covers it                | `TypeMeta`                                       |

---

## The Two Markers That Control Generation

### `+kubebuilder:object:generate=true` (package-level)

Lives in `groupversion_info.go` (above the `package` keyword).  Tells
controller-gen: **generate DeepCopy for every exported type in this package.**

Without this marker, controller-gen only generates methods for types marked with
`+kubebuilder:object:root=true` — and their `DeepCopyInto` methods will call
sub-struct methods that don't exist, causing a compile failure.

This is exactly what broke our project: the original `zz_generated.deepcopy.go`
had 68 lines (only root objects), so `in.Spec.DeepCopyInto(&out.Spec)` called a
method that didn't exist.

```go
// groupversion_info.go
// +kubebuilder:object:generate=true   ← THIS is what was missing
// +groupName=kubeschedule.io

package v1alpha1
```

### `+kubebuilder:object:root=true` (type-level)

Marks a type as a Kubernetes API root object.  Tells controller-gen to also
generate `DeepCopyObject()` so the type satisfies `runtime.Object`.

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ScheduledAction struct { ... }
```

---

## The Build Tag Guard

Every generated file starts with:

```go
//go:build !ignore_autogenerated
```

This lets controller-gen **exclude the generated file when loading the
package**.  controller-gen uses `--load-build-tags=ignore_autogenerated` by
default, so it parses your types without seeing the old generated code.
This avoids circular dependency issues (the generated code references types,
and the types don't compile without the generated code).

---

## The Call Graph

When the controller cache clones a `ScheduledAction`, the call chain looks like
this:

```
ScheduledAction.DeepCopyObject()
  └─ ScheduledAction.DeepCopy()
       └─ ScheduledAction.DeepCopyInto(out)
            ├─ *out = *in                          (TypeMeta — value types only)
            ├─ ObjectMeta.DeepCopyInto(&out..)     (k8s built-in, handles labels/annotations/etc.)
            ├─ ScheduledActionSpec.DeepCopyInto(out.Spec)
            │    ├─ *out = *in                      (Schedule — string)
            │    ├─ new(string) for TimeZone        (pointer clone)
            │    ├─ ActionSpec.DeepCopyInto(out.Action)
            │    │    ├─ TargetRef.DeepCopyInto(out.TargetRef)
            │    │    │    └─ new(string) for Namespace
            │    │    └─ make(map[string]string) for Params
            │    └─ new(bool) for Suspend
            └─ ScheduledActionStatus.DeepCopyInto(out.Status)
                 ├─ (*in.LastRun).DeepCopy()        (metav1.Time has its own DeepCopy)
                 ├─ (*in.NextRun).DeepCopy()
                 └─ make([]ExecutionHistory) + loop
                      └─ ExecutionHistory.DeepCopyInto(out)
                           ├─ Timestamp.DeepCopyInto (metav1.Time)
                           └─ new(metav1.Duration) for Duration
```

---

## Regenerating

When you change `types.go`, you **must** regenerate.  The Makefile target:

```bash
make generate
```

Which runs:

```bash
go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.21.0 \
  object crd paths=./api/v1alpha1/... output:crd:dir=./config/crd
```

This regenerates both `zz_generated.deepcopy.go` and the CRD YAML in one pass.

---

## Common Pitfalls

| Pitfall | Symptom | Fix |
|---------|---------|-----|
| Missing `+kubebuilder:object:generate=true` | Compile error: `type X has no field or method DeepCopyInto` | Add the marker above the `package` line in `groupversion_info.go` |
| Forgot to run `make generate` after changing types | Same compile error, or stale CRD | Run `make generate` |
| CRLF line endings on Windows | controller-gen silently generates incomplete output | Use `sed -i "s/\r$//"` or configure git with `core.autocrlf=input` |
| Mutating a cached object without copying | Data race panic at runtime, flaky tests | Always call `.DeepCopy()` before mutating |
| Adding a new struct but not re-running generate | Missing methods for the new struct | Run `make generate` after any type change |

---

## Key Takeaway

You never write DeepCopy by hand.  You define Go structs, add two markers
(`generate=true` at package level, `root=true` on API objects), and run
`controller-gen`.  The tool inspects every field, picks the right cloning
strategy, and writes `zz_generated.deepcopy.go`.  Your only job is to
**regenerate every time you touch `types.go`**.
