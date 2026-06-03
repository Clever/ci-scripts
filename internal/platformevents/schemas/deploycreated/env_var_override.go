package deploycreated

type EnvVarOverride struct {
    Name string `json:"name,omitempty"`
    Value string `json:"value,omitempty"`
}

func (e *EnvVarOverride) SetName(name string) {
    e.Name = name
}

func (e *EnvVarOverride) SetValue(value string) {
    e.Value = value
}
