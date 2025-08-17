package commands

import (
	"fmt"
	"strings"

	"mitl/internal/detector"
)

// Inspect analyzes project and prints summary + generated Dockerfile.
// This command provides detailed information about the detected project type and shows
// the Dockerfile that would be generated for the project.
func Inspect(args []string) error {
	detectorInstance := detector.NewProjectDetector("")
	_ = detectorInstance.Detect()

	fmt.Println("=== Project Analysis ===")
	fmt.Printf("Type: %s\n", detectorInstance.Type)
	if detectorInstance.Framework != "" {
		fmt.Printf("Framework: %s %s\n", detectorInstance.Framework, detectorInstance.Version)
	}

	if detectorInstance.Dependencies.PHP.Version != "" {
		fmt.Println("\nPHP Dependencies:")
		fmt.Printf("  Version: %s\n", detectorInstance.Dependencies.PHP.Version)
		fmt.Printf("  Extensions: %s\n", strings.Join(detectorInstance.Dependencies.PHP.Extensions, ", "))
	}

	if detectorInstance.Dependencies.Node.Version != "" {
		fmt.Println("\nNode Dependencies:")
		fmt.Printf("  Version: %s\n", detectorInstance.Dependencies.Node.Version)
		fmt.Printf("  Package Manager: %s\n", detectorInstance.Dependencies.Node.PackageManager)
	}

	generator := NewDockerfileGenerator(detectorInstance)
	dockerfile, err := generator.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	fmt.Println("\n=== Generated Dockerfile ===")
	fmt.Println(dockerfile)
	return nil
}
