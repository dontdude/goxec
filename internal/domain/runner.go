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

// Job represents a unit of work to be executed.
// It carries the Code and Language payload, along with a channel to report the result.
type Job struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Language string `json:"language"`

	// RawID is the internal Stream ID from Redis (e.g. 1700000-0).
	// We need this to Acknowledge the message later. 
	RawID string `json:"-"`

	// ResultCh is where the worker sends the execution result.
	// It is a send only channel (chan<-) to ensure the worker cannot read from it.
	ResultCh chan<- JobResult `json:"-"`
}

// JobResult encapsulates the result of a job execution.
type JobResult struct {
	JobID  string `json:"job_id"`
	Output string `json:"output"`
	Error  error  `json:"-"`
}