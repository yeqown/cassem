package testutil

import (
	"context"
	"net"
	"strings"
	"time"
)

const Podman = "podman"

type ContainerOptions struct {
	Name     string
	Hostname string
	Alias    string
	Image    string
	Network  string
	Args     []string
	Env      map[string]string
	Ports    map[string]string
	Volumes  map[string]string
}

func RequirePodman(t TB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := Run(ctx, "", Podman, "version"); err != nil {
		t.Skipf("podman is required for integration tests: %v", err)
	}
}

func BuildImage(t TB, repoRoot string, imageName string, dockerfile string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if _, err := Run(ctx, repoRoot, Podman, "build", "-t", imageName, "-f", dockerfile, "."); err != nil {
		t.Fatalf("build image %s: %v", imageName, err)
	}
}

func CreateNetwork(t TB, name string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := Run(ctx, "", Podman, "network", "create", name); err != nil {
		t.Fatalf("create podman network %s: %v", name, err)
	}
}

func RemoveNetwork(t TB, name string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, _ = Run(ctx, "", Podman, "network", "rm", name)
}

func RunContainer(t TB, opts ContainerOptions) string {
	t.Helper()

	args := []string{"run", "-d", "--name", opts.Name}
	if opts.Hostname != "" {
		args = append(args, "--hostname", opts.Hostname)
	}
	if opts.Network != "" {
		args = append(args, "--network", opts.Network)
	}
	if opts.Alias != "" {
		args = append(args, "--network-alias", opts.Alias)
	}
	for host, container := range opts.Ports {
		args = append(args, "-p", host+":"+container)
	}
	for host, container := range opts.Volumes {
		args = append(args, "-v", host+":"+container)
	}
	for key, val := range opts.Env {
		args = append(args, "-e", key+"="+val)
	}
	args = append(args, opts.Image)
	args = append(args, opts.Args...)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := Run(ctx, "", Podman, args...)
	if err != nil {
		t.Fatalf("run container %s: %v", opts.Name, err)
	}

	return strings.TrimSpace(out)
}

func StopAndRemoveContainer(t TB, id string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, _ = Run(ctx, "", Podman, "rm", "-f", id)
}

func ContainerLogs(t TB, id string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := Run(ctx, "", Podman, "logs", id)
	if err != nil {
		return err.Error()
	}
	return out
}

func FreePort(t TB) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}
	defer ln.Close()

	return ln.Addr().(*net.TCPAddr).Port
}
