package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"anchor/internal/config"
	"anchor/internal/kubecfg"
	"anchor/internal/picker"

	"github.com/spf13/cobra"
)

var projectImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Create a project from an existing kubeconfig context",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runProjectImport(cmd); err != nil {
			exitErr(err)
		}
	},
}

func runProjectImport(cmd *cobra.Command) error {
	kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
	if kubeconfig == "" {
		kubeconfig = kubecfg.DefaultKubeconfigPath()
	}
	contexts, err := kubecfg.ListContexts(kubeconfig)
	if err != nil {
		return err
	}
	if len(contexts) == 0 {
		return fmt.Errorf("no contexts in %s", kubeconfig)
	}
	labels := make([]string, len(contexts))
	for i, c := range contexts {
		labels[i] = fmt.Sprintf("%s (cluster=%s ns=%s)", c.Name, c.Cluster, c.Namespace)
	}
	picked, err := picker.Choose("Select context to import:", labels)
	if err != nil {
		return err
	}
	ctxName := strings.Split(picked, " ")[0]

	reader := bufio.NewReader(os.Stdin)
	ask := func(prompt, def string) (string, error) {
		if def != "" {
			fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, def)
		} else {
			fmt.Fprintf(os.Stderr, "%s: ", prompt)
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return def, nil
		}
		return line, nil
	}

	defaultName := ctxName
	name, err := ask("Project name", defaultName)
	if err != nil {
		return err
	}
	profile, err := ask("AWS profile", name)
	if err != nil {
		return err
	}
	region, err := ask("AWS region", "us-east-1")
	if err != nil {
		return err
	}
	cluster, err := ask("EKS cluster name", "")
	if err != nil {
		return err
	}
	tier, _ := ask("Tier", "dev")
	ns := ""
	for _, c := range contexts {
		if c.Name == ctxName {
			ns = c.Namespace
			break
		}
	}
	ns, _ = ask("Default namespace", ns)

	p := &config.Project{
		Name:             name,
		AWSProfile:       profile,
		Region:           region,
		Cluster:          cluster,
		ContextAlias:     ctxName,
		DefaultNamespace: ns,
		Tier:             tier,
	}
	if err := config.SaveProject(p); err != nil {
		return err
	}
	if err := kubecfg.CopyContextToProject(kubeconfig, ctxName, name); err != nil {
		return fmt.Errorf("saved project but kubeconfig copy failed: %w", err)
	}
	fmt.Printf("✓ Imported project %q from context %q\n", name, ctxName)
	fmt.Printf("  Run: anchor project use %s\n", name)
	return nil
}

func init() {
	projectImportCmd.Flags().String("kubeconfig", "", "Source kubeconfig (default ~/.kube/config)")
}
