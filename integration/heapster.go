package integration

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/apprenda/kismatic/pkg/retry"
)

func verifyHeapster(master NodeDeets, sshKey string) error {
	// create volumes for alertmanager, prometheus-server and grafana
	for n := 0; n < 1; n++ {
		cmd := exec.Command("./kismatic", "volume", "add", "1", "-f", "kismatic-testing.yaml")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error adding volume: %v", err)
		}
	}

	// copy PVCs
	pvcs := []string{"influxdb-pvc.yaml"}
	for _, f := range pvcs {
		if err := copyFileToRemote(fmt.Sprintf("test-resources/heapster/%s", f), fmt.Sprintf("/tmp/%s", f), master, sshKey, 1*time.Minute); err != nil {
			return fmt.Errorf("error copying %s: %v", f, err)
		}
	}

	// create PVCs
	for _, f := range pvcs {
		if err := runViaSSH([]string{fmt.Sprintf("sudo kubectl apply -f /tmp/%s", f)}, []NodeDeets{master}, sshKey, 1*time.Minute); err != nil {
			return fmt.Errorf("error creating pvc %s: %v", f, err)
		}
	}

	// verify pods are up
	deployments := []string{"heapster-influxdb"}
	for _, d := range deployments {
		if err := retry.WithBackoff(func() error {
			return runViaSSH([]string{fmt.Sprintf("sudo kubectl get deployment %s -n kube-system -o jsonpath='{.status.availableReplicas}' | grep 1", d)}, []NodeDeets{master}, sshKey, 1*time.Minute)
		}, 10); err != nil {
			return fmt.Errorf("error verifying deployment %s: %v", d, err)
		}
	}

	return nil
}
