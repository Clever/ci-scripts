package deploycreated

type Detail struct {
    App string `json:"app,omitempty"`
    ClusterEnvironment string `json:"clusterEnvironment,omitempty"`
    Environment string `json:"environment,omitempty"`
    Overrides DeployOverrides `json:"overrides,omitempty"`
    Repo string `json:"repo,omitempty"`
    TargetRevision string `json:"targetRevision,omitempty"`
    User User `json:"user,omitempty"`
}

func (d *Detail) SetApp(app string) {
    d.App = app
}

func (d *Detail) SetClusterEnvironment(clusterEnvironment string) {
    d.ClusterEnvironment = clusterEnvironment
}

func (d *Detail) SetEnvironment(environment string) {
    d.Environment = environment
}

func (d *Detail) SetOverrides(overrides DeployOverrides) {
    d.Overrides = overrides
}

func (d *Detail) SetRepo(repo string) {
    d.Repo = repo
}

func (d *Detail) SetTargetRevision(targetRevision string) {
    d.TargetRevision = targetRevision
}

func (d *Detail) SetUser(user User) {
    d.User = user
}
