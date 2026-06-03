package deploycreated

type AutoScalingConfig struct {
    Max int32 `json:"max,omitempty"`
    Min int32 `json:"min,omitempty"`
}

func (a *AutoScalingConfig) SetMax(max int32) {
    a.Max = max
}

func (a *AutoScalingConfig) SetMin(min int32) {
    a.Min = min
}
