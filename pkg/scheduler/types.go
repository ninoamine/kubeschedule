package scheduler

import (
	"errors"
	"fmt"
)

var (
	ErrNoNodes         = errors.New("no nodes provided")
	ErrNoEligibleNodes = errors.New("no nodes passed all filters")
)

// SimpleScheduler filters nodes, scores survivors, and picks the highest scorer.
type SimpleScheduler struct {
	Filters []Filter
	Scorers []Scorer
}

func (s *SimpleScheduler) Schedule(pod Pod, nodes []Node) (Node, error) {
	if len(nodes) == 0 {
		return Node{}, ErrNoNodes
	}

	eligible := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		passed := true
		for _, f := range s.Filters {
			if !f.Filter(pod, node) {
				passed = false
				break
			}
		}
		if passed {
			eligible = append(eligible, node)
		}
	}

	if len(eligible) == 0 {
		return Node{}, fmt.Errorf("%w: pod %s/%s", ErrNoEligibleNodes, pod.Namespace, pod.Name)
	}

	var bestNode Node
	bestScore := -1
	for _, node := range eligible {
		total := 0
		for _, scorer := range s.Scorers {
			total += scorer.Score(pod, node)
		}
		if total > bestScore {
			bestScore = total
			bestNode = node
		}
	}

	return bestNode, nil
}
