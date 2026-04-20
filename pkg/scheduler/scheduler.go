package scheduler

type Pod struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

type Node struct {
	Name              string
	Labels            map[string]string
	AllocatableCPU    int64
	AllocatableMemory int64
	UsedCPU           int64
	UsedMemory        int64
}

type Scheduler interface {
	Schedule(pod Pod, nodes []Node) (Node, error)
}

type Filter interface {
	Name() string
	Filter(pod Pod, node Node) bool
}

type Scorer interface {
	Name() string
	Score(pod Pod, node Node) int
}
