package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Detect docker compose command
	var dockerComposeCmd []string
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		dockerComposeCmd = []string{"docker", "compose"}
	} else if _, err := exec.LookPath("docker-compose"); err == nil {
		dockerComposeCmd = []string{"docker-compose"}
	} else {
		fmt.Fprintf(os.Stderr, "Error: docker compose not found\n")
		os.Exit(1)
	}

	// Check if required services are running
	// Use docker compose ps --services --filter to get running services
	checkCmd := exec.Command(dockerComposeCmd[0], append(dockerComposeCmd[1:], "ps", "--services", "--filter", "status=running")...)
	output, err := checkCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to check docker compose services: %v\n", err)
		fmt.Fprintf(os.Stderr, "Start services with: %s up backend postgres caddy -d\n", strings.Join(dockerComposeCmd, " "))
		os.Exit(1)
	}

	outputStr := strings.TrimSpace(string(output))
	runningServices := strings.Split(outputStr, "\n")
	
	// Convert to map for easy lookup
	servicesMap := make(map[string]bool)
	for _, svc := range runningServices {
		if svc != "" {
			servicesMap[svc] = true
		}
	}

	hasBackend := servicesMap["backend"]
	hasPostgres := servicesMap["postgres"]

	if !hasBackend || !hasPostgres {
		var missing []string
		if !hasBackend {
			missing = append(missing, "backend")
		}
		if !hasPostgres {
			missing = append(missing, "postgres")
		}
		fmt.Fprintf(os.Stderr, "Error: required services (%s) are not running\n", strings.Join(missing, ", "))
		fmt.Fprintf(os.Stderr, "Start services with: %s up backend postgres caddy -d\n", strings.Join(dockerComposeCmd, " "))
		os.Exit(1)
	}

	// Run lazyadmin in container
	runCmd := exec.Command(dockerComposeCmd[0], append(dockerComposeCmd[1:], "run", "--rm", "lazyadmin")...)
	runCmd.Stdin = os.Stdin
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	if err := runCmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: failed to run lazyadmin: %v\n", err)
		os.Exit(1)
	}
}

