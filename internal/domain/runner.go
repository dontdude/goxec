package domain

import "context"

// Output represents the result of an isolated code execution.
// It encapsulates the standard output and potential metadata.
type Output struct {
	Result string
}

// ContainerRunner defines the contract for executing code within an isolated container environment.
// Implementations of this interface handle the low-level container lifecycle management.
type ContainerRunner interface {
	// Run executes the provided code snippet in a container for the specified language.
	// It returns the execution output or an error if the container fails to start or run.
	Run(ctx context.Context, code string, language string) (string, error)
}
