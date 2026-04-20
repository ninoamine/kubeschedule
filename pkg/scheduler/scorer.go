package scheduler

type SpreadScorer struct{}

func (s SpreadScorer) Name() string { return "SpreadScorer" }

func (s SpreadScorer) Score(_ Pod, node Node) int {
	if node.AllocatableCPU == 0 {
		return 0
	}

	usedRatio := float64(node.UsedCPU) / float64(node.AllocatableCPU)
	score := int((1.0 - usedRatio) * 100)

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

type LabelPreferenceScorer struct {
	PreferredLabels map[string]string
}

func (s LabelPreferenceScorer) Name() string { return "LabelPreferenceScorer" }

func (s LabelPreferenceScorer) Score(_ Pod, node Node) int {
	if len(s.PreferredLabels) == 0 {
		return 50
	}
	matches := 0

	for k, v := range s.PreferredLabels {
		if node.Labels[k] == v {
			matches++
		}
	}
	return (matches * 100) / len(s.PreferredLabels)
}
