package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

func readYAML(filename string) (*Manifest, error) {
	// Read the file content
	d, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Parse YAML into struct
	var m Manifest
	err = yaml.Unmarshal(d, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func findFolders() ([]string, error) {
	// Find all directories containing kustomization.yaml
	var folders []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// If the current file is kustomization.yaml, add its directory to the list
		if info.Name() == "kustomization.yaml" {
			m, e := readYAML(path)
			if e != nil {
				return fmt.Errorf("failed to read yaml file %s: %w", path, e)
			}
			// Only add folder containing a Kustomization Api Kind
			if m.Kind == "Kustomization" {
				dir := filepath.Dir(path)
				folders = append(folders, dir)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error finding kustomize folders: %w", err)
	}

	// Sort the folders
	sort.Strings(folders)

	return folders, nil
}

func validate(folder string) bool {
	indent := "60"
	// Build the kustomization.yaml with kustomize
	cmd := exec.Command("kustomize", "build", folder)
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running kustomize build on %s: %v\n", folder, err)
		fmt.Fprintf(os.Stderr, "Running kustomize build on: %-"+indent+"s ERROR\n", folder+"...")
		return false
	}
	fmt.Printf("Running kustomize build on: %-"+indent+"s OK\n", folder+"...")
	return true
}

func ValidateKustomize(workers int, failFast bool) bool {
	folders, err := findFolders()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}

	if len(folders) == 0 {
		fmt.Println("No kustomize folder found")
		return true
	}

	var wg sync.WaitGroup
	taskChan := make(chan string, len(folders)) // Buffered channel for tasks

	// Start worker goroutines
	valid := true
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for folder := range taskChan { // Process tasks from channel
				if !validate(folder) {
					valid = false
					if failFast {
						os.Exit(1)
					}
				}
			}
		}()
	}

	// Send tasks to workers
	for _, folder := range folders {
		taskChan <- folder
	}
	close(taskChan) // Close channel to signal workers no more tasks are coming

	// Wait for all workers to finish
	wg.Wait()

	return valid
}
