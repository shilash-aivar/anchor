package awsx

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// EKSClusterPrivateOnly returns true when the cluster API is not publicly reachable.
func EKSClusterPrivateOnly(profile, region, cluster string) (bool, error) {
	args := []string{
		"eks", "describe-cluster",
		"--name", cluster,
		"--region", region,
		"--output", "json",
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	out, err := exec.Command("aws", args...).Output()
	if err != nil {
		return false, fmt.Errorf("describe cluster %q: %w", cluster, err)
	}
	var resp struct {
		Cluster struct {
			ResourcesVpcConfig struct {
				EndpointPublicAccess  bool `json:"endpointPublicAccess"`
				EndpointPrivateAccess bool `json:"endpointPrivateAccess"`
			} `json:"resourcesVpcConfig"`
		} `json:"cluster"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return false, err
	}
	cfg := resp.Cluster.ResourcesVpcConfig
	if !cfg.EndpointPublicAccess && cfg.EndpointPrivateAccess {
		return true, nil
	}
	return false, nil
}
