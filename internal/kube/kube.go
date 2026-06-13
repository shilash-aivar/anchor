package kube

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type PodLine struct {
	Name       string
	Ready      string
	Status     string
	Restarts   string
	Containers []string
}

func KubectlAvailable() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}

func runKubectl(kubeconfig, context, namespace string, args ...string) *exec.Cmd {
	full := append([]string{}, args...)
	cmd := exec.Command("kubectl", full...)
	env := os.Environ()
	if kubeconfig != "" {
		env = append(env, "KUBECONFIG="+kubeconfig)
	}
	cmd.Env = env
	if context != "" {
		cmd.Args = append([]string{"kubectl", "--context", context}, full...)
	}
	if namespace != "" {
		insertAt := 1
		if context != "" {
			insertAt = 3
		}
		newArgs := append([]string{}, cmd.Args[:insertAt]...)
		newArgs = append(newArgs, "-n", namespace)
		newArgs = append(newArgs, cmd.Args[insertAt:]...)
		cmd.Args = newArgs
	}
	return cmd
}

func captureKubectl(kubeconfig, context, namespace string, args ...string) (string, error) {
	cmd := runKubectl(kubeconfig, context, namespace, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(out.String()), err)
	}
	return strings.TrimSpace(out.String()), nil
}

func UseNamespace(kubeconfig, context, namespace string) error {
	cmd := runKubectl(kubeconfig, context, "", "config", "set-context", context, "--namespace="+namespace)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set namespace: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func CurrentContext(kubeconfig string) (string, error) {
	cmd := exec.Command("kubectl", "config", "current-context")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func ListNamespaces(kubeconfig, context string) ([]string, error) {
	out, err := captureKubectl(kubeconfig, context, "", "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Fields(out), nil
}

func ListPodLines(kubeconfig, context, namespace string) ([]PodLine, error) {
	out, err := captureKubectl(kubeconfig, context, namespace, "get", "pods", "--no-headers")
	if err != nil {
		return nil, err
	}
	var lines []PodLine
	for _, row := range strings.Split(out, "\n") {
		if row == "" {
			continue
		}
		f := strings.Fields(row)
		if len(f) < 4 {
			continue
		}
		pl := PodLine{Name: f[0], Ready: f[1], Status: f[2], Restarts: f[3]}
		containers, _ := ListContainers(kubeconfig, context, namespace, pl.Name)
		pl.Containers = containers
		lines = append(lines, pl)
	}
	return lines, nil
}

func ListPods(kubeconfig, context, namespace string) ([]string, error) {
	lines, err := ListPodLines(kubeconfig, context, namespace)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(lines))
	for i, l := range lines {
		names[i] = l.Name
	}
	return names, nil
}

func ListContainers(kubeconfig, context, namespace, pod string) ([]string, error) {
	out, err := captureKubectl(kubeconfig, context, namespace, "get", "pod", pod, "-o", "jsonpath={.spec.containers[*].name}")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Fields(out), nil
}

func PickContainer(containers []string, requested string) (string, error) {
	if requested != "" {
		return requested, nil
	}
	if len(containers) <= 1 {
		if len(containers) == 1 {
			return containers[0], nil
		}
		return "", nil
	}
	return "", fmt.Errorf("pod has multiple containers — use -c <name> (%s)", strings.Join(containers, ", "))
}

func ExecPod(kubeconfig, context, namespace, pod, container string, cmdArgs []string) error {
	args := []string{"exec", "-it", pod}
	if container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, "--")
	args = append(args, cmdArgs...)
	c := runKubectl(kubeconfig, context, namespace, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func Passthrough(kubeconfig, context, namespace string, args []string) error {
	c := runKubectl(kubeconfig, context, namespace, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func ClusterReachable(kubeconfig, context string) error {
	_, err := captureKubectl(kubeconfig, context, "", "cluster-info")
	return err
}

func PortForward(kubeconfig, context, namespace, target string, ports []string) error {
	args := append([]string{"port-forward", target}, ports...)
	return Passthrough(kubeconfig, context, namespace, args)
}

func RolloutWatch(kubeconfig, context, namespace, resource string) error {
	return Passthrough(kubeconfig, context, namespace, []string{"rollout", "status", resource, "-w"})
}

func GetWatch(kubeconfig, context, namespace string, args []string) error {
	full := append(args, "-w")
	return Passthrough(kubeconfig, context, namespace, full)
}

func Events(kubeconfig, context, namespace string, warningsOnly bool) error {
	args := []string{"get", "events", "--sort-by=.lastTimestamp"}
	if warningsOnly {
		args = append(args, "--field-selector", "type!=Normal")
	}
	return Passthrough(kubeconfig, context, namespace, args)
}

func CopyFromPod(kubeconfig, context, namespace, pod, container, remote, local string) error {
	args := []string{"cp"}
	if container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, fmt.Sprintf("%s/%s:%s", namespace, pod, remote), local)
	c := exec.Command("kubectl", args...)
	c.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	if context != "" {
		c.Args = append([]string{"kubectl", "--context", context}, args...)
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func CopyToPod(kubeconfig, context, namespace, pod, container, local, remote string) error {
	args := []string{"cp"}
	if container != "" {
		args = append(args, "-c", container)
	}
	args = append(args, local, fmt.Sprintf("%s/%s:%s", namespace, pod, remote))
	c := exec.Command("kubectl", args...)
	c.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	if context != "" {
		c.Args = append([]string{"kubectl", "--context", context}, args...)
	}
	return c.Run()
}

func FindResourcesSimple(kubeconfig, context, namespace, query string) (string, error) {
	out, err := captureKubectl(kubeconfig, context, namespace, "get", "pods,deployments,svc", "--no-headers")
	if err != nil {
		return "", err
	}
	var matched []string
	q := strings.ToLower(query)
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if strings.Contains(strings.ToLower(line), q) {
			matched = append(matched, line)
		}
	}
	return strings.Join(matched, "\n"), nil
}

func DebugPod(kubeconfig, context, namespace, pod, container string, args []string) error {
	kargs := []string{"debug", pod, "-it"}
	if container != "" {
		kargs = append(kargs, "-c", container)
	}
	kargs = append(kargs, args...)
	return Passthrough(kubeconfig, context, namespace, kargs)
}

func LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func RunExternal(name string, args []string, env map[string]string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RunHelm(kubeconfig, context, namespace string, args []string) error {
	if _, err := LookPath("helm"); err != nil {
		return fmt.Errorf("helm not found — install: brew install helm")
	}
	hasNS := false
	for _, a := range args {
		if a == "-n" || a == "--namespace" {
			hasNS = true
			break
		}
	}
	full := append([]string{}, args...)
	if namespace != "" && !hasNS {
		full = append(full, "-n", namespace)
	}
	cmd := exec.Command("helm", full...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	_ = context
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func SessionEnvMap(kubeconfig, context, namespace string, extra map[string]string) map[string]string {
	m := map[string]string{"KUBECONFIG": kubeconfig}
	for k, v := range extra {
		m[k] = v
	}
	_ = context
	_ = namespace
	return m
}
