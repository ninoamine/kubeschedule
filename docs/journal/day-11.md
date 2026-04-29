# Day 11 — ScheduledAction CRD Field Design

## What Is a CRD?

A Custom Resource Definition (CRD) extends the Kubernetes API with a new
resource type. Once installed, the API server validates, stores, and serves the
custom resource exactly like built-in resources (Pods, Deployments, …). CRDs
give us:

- **kubectl integration** — `kubectl get scheduledactions` works out of the box.
- **Watch semantics** — controllers can watch/list custom resources through
  Informers just like native ones.
- **RBAC** — permissions are managed through standard ClusterRole / Role rules.
- **OpenAPI v3 validation** — the API server rejects invalid resources at
  admission time, before they ever reach the controller.

## Spec vs Status: The Split

Kubernetes follows a *desired state / observed state* pattern:

| Field   | Written by | Purpose                                     |
|---------|-----------|----------------------------------------------|
| `spec`  | User      | Declares the **desired** state.              |
| `status`| Controller| Reports the **observed** state.              |

Enabling the **status subresource** (`+kubebuilder:subresource:status`) creates a
dedicated `/status` endpoint so user updates to `spec` and controller updates to
`status` never race on the same resource version. This also lets RBAC grant
"update spec" and "update status" independently.

## ScheduledAction — API Group & Versioning

| Property   | Value                         |
|-----------|-------------------------------|
| Group     | `kubeschedule.io`             |
| Version   | `v1alpha1`                    |
| Kind      | `ScheduledAction`             |
| Scope     | Namespaced                    |

Starting at `v1alpha1` signals the API is experimental. We will graduate to
`v1beta1` and eventually `v1` with conversion webhooks as the schema stabilises.

## Spec Design

```go
type ScheduledActionSpec struct {
    // Cron expression (5-field standard cron format).
    // Examples: "0 22 * * 1-5" (weeknights at 22:00), "*/5 * * * *" (every 5 min).
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=9
    Schedule string `json:"schedule"`

    // IANA time-zone name applied to the cron schedule.
    // Defaults to "UTC" if omitted (set by defaulting webhook).
    // +optional
    TimeZone *string `json:"timeZone,omitempty"`

    // The action to perform on each trigger.
    // +kubebuilder:validation:Required
    Action ActionSpec `json:"action"`

    // When true the controller skips scheduling entirely.
    // +kubebuilder:default=false
    // +optional
    Suspend *bool `json:"suspend,omitempty"`

    // Max successful runs kept in status.history.
    // +kubebuilder:default=5
    // +kubebuilder:validation:Minimum=0
    // +optional
    SuccessfulHistoryLimit *int32 `json:"successfulHistoryLimit,omitempty"`

    // Max failed runs kept in status.history.
    // +kubebuilder:default=3
    // +kubebuilder:validation:Minimum=0
    // +optional
    FailedHistoryLimit *int32 `json:"failedHistoryLimit,omitempty"`
}
```

### ActionSpec

```go
type ActionSpec struct {
    // The operation to perform.
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Enum=Scale;Patch;Delete;RolloutRestart;RotateSecret;Cordon;Exec;Apply
    Type ActionType `json:"type"`

    // Reference to the target Kubernetes resource.
    // +kubebuilder:validation:Required
    TargetRef TargetRef `json:"targetRef"`

    // Free-form parameters passed to the executor (e.g. {"replicas": "0"}).
    // +optional
    Params map[string]string `json:"params,omitempty"`
}
```

### ActionType Enum

```go
// +kubebuilder:validation:Enum=Scale;Patch;Delete;RolloutRestart;RotateSecret;Cordon;Exec;Apply
type ActionType string

const (
    ActionTypeScale          ActionType = "Scale"
    ActionTypePatch          ActionType = "Patch"
    ActionTypeDelete         ActionType = "Delete"
    ActionTypeRolloutRestart ActionType = "RolloutRestart"
    ActionTypeRotateSecret   ActionType = "RotateSecret"
    ActionTypeCordon         ActionType = "Cordon"
    ActionTypeExec           ActionType = "Exec"
    ActionTypeApply          ActionType = "Apply"
)
```

### TargetRef

```go
type TargetRef struct {
    // API group and version (e.g. "apps/v1").
    // +kubebuilder:validation:Required
    APIVersion string `json:"apiVersion"`

    // Resource kind (e.g. "Deployment").
    // +kubebuilder:validation:Required
    Kind string `json:"kind"`

    // Resource name.
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Namespace of the target. Defaults to the ScheduledAction's namespace.
    // +optional
    Namespace *string `json:"namespace,omitempty"`
}
```

## Status Design

```go
type ScheduledActionStatus struct {
    // Timestamp of the most recent execution.
    // +optional
    LastRun *metav1.Time `json:"lastRun,omitempty"`

    // Computed next fire time based on the cron schedule.
    // +optional
    NextRun *metav1.Time `json:"nextRun,omitempty"`

    // Rolling window of execution results.
    // +optional
    History []ExecutionHistory `json:"history,omitempty"`

    // Standard Kubernetes conditions (Ready, Suspended, …).
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

### ExecutionHistory

```go
type ExecutionHistory struct {
    // When the execution started.
    Timestamp metav1.Time `json:"timestamp"`

    // Outcome — "Success" or "Failure".
    // +kubebuilder:validation:Enum=Success;Failure
    Result string `json:"result"`

    // Human-readable detail (error message on failure, summary on success).
    // +optional
    Message string `json:"message,omitempty"`

    // Wall-clock duration of the execution.
    // +optional
    Duration *metav1.Duration `json:"duration,omitempty"`
}
```

### Conditions

Following the standard `metav1.Condition` contract:

| Type        | Meaning                                                    |
|------------|-------------------------------------------------------------|
| `Ready`     | `True` when the cron schedule is valid and the controller is scheduling. `False` with reason `InvalidCron` when parsing fails. |
| `Suspended` | `True` when `spec.suspend` is `true`.                       |

Each condition carries `ObservedGeneration` so consumers can tell whether the
controller has processed the latest `spec` change.

## Full Example YAML

```yaml
apiVersion: kubeschedule.io/v1alpha1
kind: ScheduledAction
metadata:
  name: night-scaledown
  namespace: production
spec:
  schedule: "0 22 * * 1-5"
  timeZone: "Europe/Paris"
  action:
    type: Scale
    targetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: api-server
    params:
      replicas: "0"
  suspend: false
  successfulHistoryLimit: 5
  failedHistoryLimit: 3
status:
  lastRun: "2026-04-28T22:00:00Z"
  nextRun: "2026-04-29T22:00:00Z"
  conditions:
    - type: Ready
      status: "True"
      reason: ScheduleValid
      message: "Cron expression parsed successfully"
      lastTransitionTime: "2026-04-28T22:00:00Z"
      observedGeneration: 1
  history:
    - timestamp: "2026-04-28T22:00:00Z"
      result: Success
      message: "Scaled api-server to 0 replicas"
      duration: "1.23s"
    - timestamp: "2026-04-27T22:00:00Z"
      result: Success
      message: "Scaled api-server to 0 replicas"
      duration: "0.98s"
```

## Design Decisions & Rationale

1. **`timeZone` as pointer** — `nil` means "not set"; the defaulting webhook
   fills in `"UTC"`. Using a pointer makes the "user didn't specify" case
   explicit versus an empty string.

2. **`suspend` as `*bool`** — same reasoning; `nil` ≠ `false` at the API level,
   even though the default is `false`.

3. **`params` as `map[string]string`** — keeps the CRD schema simple and avoids
   recursive `apiextensionsv1.JSON` fields. Each executor parses the string
   values it cares about (e.g. `strconv.Atoi(params["replicas"])`).

4. **`Conditions` slice** — follows the upstream Kubernetes condition convention
   (`metav1.Condition`) so tooling like `kubectl wait --for=condition=Ready`
   works out of the box.

5. **History limits** — without bounds the status would grow unboundedly. We
   default to 5 successes and 3 failures, matching CronJob's
   `successfulJobsHistoryLimit` / `failedJobsHistoryLimit` precedent.

6. **Namespaced scope** — actions typically target resources within the same
   namespace. Cross-namespace targeting is possible via `targetRef.namespace`
   (requires RBAC in the target namespace).

7. **ActionType immutability** — changing `action.type` after creation would
   invalidate historical results. The validating webhook (Phase 5) will reject
   type changes on update.

## What's Next

- **Day 12** — Translate this design into Go types in `api/v1alpha1/types.go`.
- **Day 13–14** — Add kubebuilder markers for validation and code generation.
- **Day 15** — Run `controller-gen` to produce the CRD YAML and DeepCopy methods.

## Sources

- [Kubernetes CRD documentation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/)
- [API Conventions — spec and status](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)
- [Status subresource design](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource)
- [Condition conventions (KEP-1623)](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/1623-standardize-conditions)
- [Kubebuilder markers reference](https://book.kubebuilder.io/reference/markers)
