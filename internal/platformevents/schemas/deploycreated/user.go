package deploycreated

type User struct {
    Email string `json:"email,omitempty"`
    GithubUsername string `json:"githubUsername,omitempty"`
}

func (u *User) SetEmail(email string) {
    u.Email = email
}

func (u *User) SetGithubUsername(githubUsername string) {
    u.GithubUsername = githubUsername
}
