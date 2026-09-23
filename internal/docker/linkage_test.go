package docker

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntrypointBinary(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *container.Config
		want   string
		wantOK bool
	}{
		{name: "nil config", cfg: nil},
		{name: "no entrypoint or cmd", cfg: &container.Config{}},
		{
			name:   "cmd only",
			cfg:    &container.Config{Cmd: []string{"/root/app/app", "--flag"}},
			want:   "/root/app/app",
			wantOK: true,
		},
		{
			name: "entrypoint wins over cmd",
			cfg: &container.Config{
				Entrypoint: []string{"/bin/app"},
				Cmd:        []string{"serve"},
			},
			want:   "/bin/app",
			wantOK: true,
		},
		{
			name:   "empty entrypoint falls back to cmd",
			cfg:    &container.Config{Entrypoint: []string{""}, Cmd: []string{"./app"}},
			want:   "./app",
			wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := entrypointBinary(tt.cfg)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsMissingShell(t *testing.T) {
	assert.True(t, isMissingShell(errors.New(`exec: "/bin/sh": stat /bin/sh: no such file or directory`)))
	assert.True(t, isMissingShell(errors.New(`exec: "/bin/sh": executable file not found in $PATH`)))
	assert.False(t, isMissingShell(errors.New("Cannot connect to the Docker daemon")))
}

func TestVerifyLinkageModes(t *testing.T) {
	// Neither mode reaches the docker client, so a zero Docker is safe.
	d := &Docker{}
	assert.NoError(t, d.VerifyLinkage(context.Background(), "img", LinkageOff))
	assert.Error(t, d.VerifyLinkage(context.Background(), "img", "strict"))
}

// TestLinkageCheckScript runs the embedded in-container script against a
// fake ldd, covering each outcome the real ldd can produce.
func TestLinkageCheckScript(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app"), []byte("\x7fELFbinary"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/sh\n"), 0o755))

	tests := []struct {
		name     string
		bin      string
		lddOut   string
		lddExit  int
		wantExit int
	}{
		{name: "resolves", bin: "./app", lddOut: "libc.so.6 => /lib/libc.so.6 (0x1)"},
		{
			name:     "glibc too old",
			bin:      "./app",
			lddOut:   "./app: /lib/libc.so.6: version `GLIBC_2.38' not found (required by ./app)",
			wantExit: 1,
		},
		{name: "ldd errors", bin: "./app", lddOut: "unexpected", lddExit: 2, wantExit: 1},
		{name: "glibc static", bin: "./app", lddOut: "not a dynamic executable", lddExit: 1},
		{name: "musl static", bin: "./app", lddOut: "./app: Not a valid dynamic program", lddExit: 1},
		{name: "library name contains error", bin: "./app", lddOut: "libgpg-error.so.0 => /lib/libgpg-error.so.0"},
		{name: "non-ELF entrypoint", bin: "./script.sh"},
		{name: "missing entrypoint", bin: "./missing", wantExit: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(binDir, "out"), []byte(tt.lddOut+"\n"), 0o644))
			fakeLdd := "#!/bin/sh\ncat " + filepath.Join(binDir, "out") + "\nexit " + strconv.Itoa(tt.lddExit) + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(binDir, "ldd"), []byte(fakeLdd), 0o755))

			cmd := exec.Command("/bin/sh", "-c", linkageCheck, "linkage-check", tt.bin)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
			out, err := cmd.CombinedOutput()

			exit := 0
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exit = exitErr.ExitCode()
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantExit, exit, string(out))
		})
	}
}
