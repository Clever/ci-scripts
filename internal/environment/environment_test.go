package environment

import "testing"

func TestAuthorIsFleetBot(t *testing.T) {
	tests := []struct {
		name   string
		author string
		want   bool
	}{
		{
			name:   "fleet bot author (name + noreply email)",
			author: "clever-fleet[bot]\n316729001+clever-fleet[bot]@users.noreply.github.com",
			want:   true,
		},
		{
			name:   "human author",
			author: "Andrew Marine\nandrew.marine@clever.com",
			want:   false,
		},
		{
			name:   "other bot",
			author: "backstage-clever[bot]\n123+backstage-clever[bot]@users.noreply.github.com",
			want:   false,
		},
		{
			name:   "empty",
			author: "",
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := authorIsFleetBot(tt.author); got != tt.want {
				t.Errorf("authorIsFleetBot(%q) = %v, want %v", tt.author, got, tt.want)
			}
		})
	}
}

func TestRunOwnerFromCoAuthors(t *testing.T) {
	tests := []struct {
		name     string
		trailers string
		want     string
	}{
		{
			name:     "run owner co-author",
			trailers: "andrew.marine <andrew.marine@clever.com>",
			want:     "andrew.marine@clever.com",
		},
		{
			name:     "skips bot co-author, returns human",
			trailers: "clever-fleet[bot] <316729001+clever-fleet[bot]@users.noreply.github.com>\nandrew.marine <andrew.marine@clever.com>",
			want:     "andrew.marine@clever.com",
		},
		{
			name:     "only bot co-author -> none",
			trailers: "clever-fleet[bot] <316729001+clever-fleet[bot]@users.noreply.github.com>",
			want:     "",
		},
		{
			name:     "no trailers",
			trailers: "",
			want:     "",
		},
		{
			name:     "malformed line ignored",
			trailers: "no-angle-brackets-here\njane.doe <jane.doe@clever.com>",
			want:     "jane.doe@clever.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runOwnerFromCoAuthors(tt.trailers); got != tt.want {
				t.Errorf("runOwnerFromCoAuthors(%q) = %q, want %q", tt.trailers, got, tt.want)
			}
		})
	}
}

func TestEmailBetweenAngles(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Name <addr@clever.com>", "addr@clever.com"},
		{"  Name <addr@clever.com>  ", "addr@clever.com"},
		{"no brackets", ""},
		{"<>", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := emailBetweenAngles(tt.in); got != tt.want {
			t.Errorf("emailBetweenAngles(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
