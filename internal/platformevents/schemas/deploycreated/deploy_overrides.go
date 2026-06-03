package deploycreated

type DeployOverrides struct {
    Autoscaling AutoScalingConfig `json:"autoscaling,omitempty"`
    Env []EnvVarOverride `json:"env,omitempty"`
}

func (d *DeployOverrides) SetAutoscaling(autoscaling AutoScalingConfig) {
    d.Autoscaling = autoscaling
}

func (d *DeployOverrides) SetEnv(env []EnvVarOverride) {
    d.Env = env
}
