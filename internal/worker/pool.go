package worker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/dontdude/goxec/internal/domain"
)

// Pool implements a fixed-size worker pool pattern.
// It throttles the concurrent execution of code usage using a buffered channel or fixed goroutines.
type Pool struct {
	// workerCount determines how many concurrent Docker containers can run.
	workerCount int
	// tasksCh is the queue for incoming jobs.
	tasksCh chan domain.Job
	// wg tracks active workers to ensure graceful shutdown.
	wg sync.WaitGroup
	runner domain.ContainerRunner
}

// NewPool initializes the worker pool with a fixed concurrency limit. 
func NewPool(concurrency int, runner domain.ContainerRunner) *Pool {
	return &Pool{
		workerCount: concurrency,
		// Buffer the channel to allow non-blocking submission up to a certain point.
		tasksCh: make(chan domain.Job, concurrency),
		runner: runner,
	}
}

// Start spawns the fixed number of worker goroutines.
// It returns immediately.
func (p *Pool) Start() {
	slog.Info("Starting worker pool", "concurrency", p.workerCount)

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop initiates a graceful shutdown.
// It closes the jobs channel, which signals all workers to finish their current task and exit.
// It blocks until all workers have exited. 
func (p *Pool) Stop() {
	slog.Info("Stopping worker pool, waiting for tasks to drain...")
	close(p.tasksCh)
	p.wg.Wait()
	slog.Info("Worker pool stopped")
}

// Submit adds a job to the queue.
// It blocks if the queue (and workers) are fully saturated.
func (p *Pool) Submit(job domain.Job) {
	p.tasksCh <- job
}

// worker is the core logic that runs inside a goroutine.
func (p *Pool) worker(id int) {
	defer p.wg.Done()
	slog.Info("Worker started", "workerId", id)

	// Range over the channel continuously reads jobs until the channel is closed. 
	for job := range p.tasksCh {
		slog.Debug("Processing job", "workerId", id, "jobID", job.ID)

		// Create a separate context for the execution to ensure independent timeouts
		// In a real app, you might inherit from a parent context or allow the job to specify one.
		ctx := context.Background()

		output, err := p.runner.Run(ctx, job.Code, job.Language)

		// Report result
		job.ResultCh <- domain.JobResult{
			Output: output,
			Error: err,
		}
	}

	slog.Info("Worker stopped", "workerID", id)
}