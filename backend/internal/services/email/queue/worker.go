package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Standard errors
var (
	ErrWorkerPoolClosed  = errors.New("worker pool is closed")
	ErrWorkerPoolRunning = errors.New("worker pool is already running")
)

// EmailSender is the interface for sending emails
type EmailSender interface {
	Send(ctx context.Context, job *EmailJob) error
}

// WorkerConfig holds worker pool configuration
type WorkerConfig struct {
	// Pool settings
	NumWorkers int `json:"num_workers"`
	MaxWorkers int `json:"max_workers"`
	MinWorkers int `json:"min_workers"`

	// Processing
	PollInterval    time.Duration `json:"poll_interval"`
	ProcessTimeout  time.Duration `json:"process_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// Rate limiting
	RateLimit  float64 `json:"rate_limit"`
	BurstLimit int     `json:"burst_limit"`

	// Health check
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	StaleJobInterval    time.Duration `json:"stale_job_interval"`

	// Auto-scaling
	AutoScale          bool          `json:"auto_scale"`
	ScaleUpThreshold   float64       `json:"scale_up_threshold"` // Queue length / workers
	ScaleDownThreshold float64       `json:"scale_down_threshold"`
	ScaleCooldown      time.Duration `json:"scale_cooldown"`
}

// DefaultWorkerConfig returns default configuration
func DefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		NumWorkers:          runtime.NumCPU(),
		MaxWorkers:          runtime.NumCPU() * 4,
		MinWorkers:          1,
		PollInterval:        100 * time.Millisecond,
		ProcessTimeout:      30 * time.Second,
		ShutdownTimeout:     30 * time.Second,
		RateLimit:           100,
		BurstLimit:          200,
		HealthCheckInterval: 30 * time.Second,
		StaleJobInterval:    time.Minute,
		AutoScale:           true,
		ScaleUpThreshold:    10,
		ScaleDownThreshold:  2,
		ScaleCooldown:       time.Minute,
	}
}

// WorkerPool manages a pool of email processing workers
type WorkerPool struct {
	queue       *EmailQueue
	sender      EmailSender
	config      *WorkerConfig
	rateLimiter *RateLimiter

	// State
	workers   map[string]*Worker
	workersMu sync.RWMutex
	running   atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// Metrics
	metrics *WorkerMetrics

	// Channels
	jobChan    chan *EmailJob
	resultChan chan *JobResult

	// Scaling
	lastScaleTime time.Time
	scaleMu       sync.Mutex

	// Callbacks
	onJobStart    func(*EmailJob)
	onJobComplete func(*EmailJob, *JobResult)
	onJobFail     func(*EmailJob, error)
	onWorkerStart func(*Worker)
	onWorkerStop  func(*Worker)
}

// Worker represents a single worker in the pool
type Worker struct {
	ID        string
	pool      *WorkerPool
	ctx       context.Context
	cancel    context.CancelFunc
	started   time.Time
	processed atomic.Int64
	failed    atomic.Int64
	lastJob   atomic.Value // *EmailJob
	state     atomic.Value // WorkerState
}

// WorkerState represents the state of a worker
type WorkerState string

const (
	WorkerStateIdle       WorkerState = "idle"
	WorkerStateProcessing WorkerState = "processing"
	WorkerStateStopping   WorkerState = "stopping"
	WorkerStateStopped    WorkerState = "stopped"
)

// JobResult represents the result of processing a job
type JobResult struct {
	JobID      string
	Success    bool
	Error      error
	Duration   time.Duration
	WorkerID   string
	RetryCount int
}

// WorkerMetrics holds worker pool metrics
type WorkerMetrics struct {
	mu sync.RWMutex

	// Counters
	JobsProcessed int64
	JobsSucceeded int64
	JobsFailed    int64
	JobsRetried   int64

	// Gauges
	ActiveWorkers int32
	QueueDepth    int64

	// Histograms
	ProcessingTimes []time.Duration
	maxSamples      int

	// Rates
	lastMinuteJobs int64
	lastMinuteTime time.Time
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(queue *EmailQueue, sender EmailSender, config *WorkerConfig) *WorkerPool {
	if config == nil {
		config = DefaultWorkerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		queue:       queue,
		sender:      sender,
		config:      config,
		rateLimiter: NewRateLimiter(config.RateLimit, config.BurstLimit),
		workers:     make(map[string]*Worker),
		ctx:         ctx,
		cancel:      cancel,
		metrics:     newWorkerMetrics(),
		jobChan:     make(chan *EmailJob, config.NumWorkers*2),
		resultChan:  make(chan *JobResult, config.NumWorkers*2),
	}

	return pool
}

// newWorkerMetrics creates new worker metrics
func newWorkerMetrics() *WorkerMetrics {
	return &WorkerMetrics{
		ProcessingTimes: make([]time.Duration, 0, 1000),
		maxSamples:      1000,
		lastMinuteTime:  time.Now(),
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start() error {
	if p.running.Swap(true) {
		return ErrWorkerPoolRunning
	}

	// Start initial workers
	for i := 0; i < p.config.NumWorkers; i++ {
		p.addWorker()
	}

	// Start dispatcher
	p.wg.Add(1)
	go p.dispatcher()

	// Start result handler
	p.wg.Add(1)
	go p.resultHandler()

	// Start health checker
	p.wg.Add(1)
	go p.healthChecker()

	// Start stale job recovery
	p.wg.Add(1)
	go p.staleJobRecovery()

	// Start auto-scaler if enabled
	if p.config.AutoScale {
		p.wg.Add(1)
		go p.autoScaler()
	}

	return nil
}

// Stop gracefully stops the worker pool
func (p *WorkerPool) Stop() error {
	if !p.running.Swap(false) {
		return nil
	}

	// Cancel context to signal shutdown
	p.cancel()

	// Wait for shutdown with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(p.config.ShutdownTimeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// dispatcher fetches jobs from the queue and dispatches to workers
func (p *WorkerPool) dispatcher() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.dispatchJobs()
		}
	}
}

// dispatchJobs fetches and dispatches available jobs
func (p *WorkerPool) dispatchJobs() {
	// Check rate limit
	if !p.rateLimiter.Allow() {
		return
	}

	// Get available worker
	worker := p.getIdleWorker()
	if worker == nil {
		return
	}

	// Dequeue job
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	job, err := p.queue.Dequeue(ctx, worker.ID)
	if err != nil {
		if !errors.Is(err, ErrQueueClosed) {
			log.Printf("Error dequeuing job: %v", err)
		}
		return
	}

	if job == nil {
		return
	}

	// Send job to worker
	select {
	case p.jobChan <- job:
		p.metrics.recordJobDispatched()
	default:
		// Channel full, put job back
		p.queue.Fail(ctx, job.ID, fmt.Errorf("worker channel full"))
	}
}

// resultHandler processes job results
func (p *WorkerPool) resultHandler() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case result := <-p.resultChan:
			p.handleResult(result)
		}
	}
}

// handleResult processes a job result
func (p *WorkerPool) handleResult(result *JobResult) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p.metrics.recordJobCompleted(result)

	if result.Success {
		if err := p.queue.Complete(ctx, result.JobID); err != nil {
			log.Printf("Error completing job %s: %v", result.JobID, err)
		}
	} else {
		if err := p.queue.Fail(ctx, result.JobID, result.Error); err != nil {
			log.Printf("Error failing job %s: %v", result.JobID, err)
		}
	}
}

// healthChecker monitors worker health
func (p *WorkerPool) healthChecker() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.checkWorkerHealth()
		}
	}
}

// checkWorkerHealth checks and repairs unhealthy workers
func (p *WorkerPool) checkWorkerHealth() {
	p.workersMu.RLock()
	workers := make([]*Worker, 0, len(p.workers))
	for _, w := range p.workers {
		workers = append(workers, w)
	}
	p.workersMu.RUnlock()

	for _, worker := range workers {
		state := worker.state.Load().(WorkerState)

		// Check for stuck workers
		if state == WorkerStateProcessing {
			if lastJob := worker.lastJob.Load(); lastJob != nil {
				job := lastJob.(*EmailJob)
				if job.StartedAt != nil && time.Since(*job.StartedAt) > p.config.ProcessTimeout*2 {
					log.Printf("Worker %s appears stuck, restarting", worker.ID)
					p.restartWorker(worker)
				}
			}
		}
	}
}

// staleJobRecovery recovers stale jobs
func (p *WorkerPool) staleJobRecovery() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.StaleJobInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			recovered, err := p.queue.RecoverStaleJobs(ctx)
			cancel()

			if err != nil {
				log.Printf("Error recovering stale jobs: %v", err)
			} else if recovered > 0 {
				log.Printf("Recovered %d stale jobs", recovered)
			}
		}
	}
}

// autoScaler automatically adjusts worker count
func (p *WorkerPool) autoScaler() {
	defer p.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.scaleWorkers()
		}
	}
}

// scaleWorkers adjusts worker count based on load
func (p *WorkerPool) scaleWorkers() {
	p.scaleMu.Lock()
	defer p.scaleMu.Unlock()

	// Check cooldown
	if time.Since(p.lastScaleTime) < p.config.ScaleCooldown {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queueSize, err := p.queue.Size(ctx)
	if err != nil {
		return
	}

	p.workersMu.RLock()
	workerCount := len(p.workers)
	p.workersMu.RUnlock()

	if workerCount == 0 {
		return
	}

	ratio := float64(queueSize) / float64(workerCount)

	// Scale up
	if ratio > p.config.ScaleUpThreshold && workerCount < p.config.MaxWorkers {
		newCount := min(workerCount*2, p.config.MaxWorkers)
		for i := workerCount; i < newCount; i++ {
			p.addWorker()
		}
		p.lastScaleTime = time.Now()
		log.Printf("Scaled up to %d workers (ratio: %.2f)", newCount, ratio)
	}

	// Scale down
	if ratio < p.config.ScaleDownThreshold && workerCount > p.config.MinWorkers {
		newCount := max(workerCount/2, p.config.MinWorkers)
		toRemove := workerCount - newCount

		p.workersMu.Lock()
		removed := 0
		for id, worker := range p.workers {
			if removed >= toRemove {
				break
			}
			if worker.state.Load().(WorkerState) == WorkerStateIdle {
				worker.cancel()
				delete(p.workers, id)
				removed++
			}
		}
		p.workersMu.Unlock()

		if removed > 0 {
			p.lastScaleTime = time.Now()
			log.Printf("Scaled down by %d workers (ratio: %.2f)", removed, ratio)
		}
	}

	p.metrics.ActiveWorkers = int32(workerCount)
	p.metrics.QueueDepth = queueSize
}

// addWorker adds a new worker to the pool
func (p *WorkerPool) addWorker() *Worker {
	ctx, cancel := context.WithCancel(p.ctx)

	worker := &Worker{
		ID:      generateWorkerID(),
		pool:    p,
		ctx:     ctx,
		cancel:  cancel,
		started: time.Now(),
	}
	worker.state.Store(WorkerStateIdle)

	p.workersMu.Lock()
	p.workers[worker.ID] = worker
	p.workersMu.Unlock()

	p.wg.Add(1)
	go worker.run()

	atomic.AddInt32(&p.metrics.ActiveWorkers, 1)

	if p.onWorkerStart != nil {
		p.onWorkerStart(worker)
	}

	return worker
}

// removeWorker removes a worker from the pool
func (p *WorkerPool) removeWorker(workerID string) {
	p.workersMu.Lock()
	worker, exists := p.workers[workerID]
	if exists {
		delete(p.workers, workerID)
	}
	p.workersMu.Unlock()

	if exists && worker != nil {
		worker.cancel()
		atomic.AddInt32(&p.metrics.ActiveWorkers, -1)

		if p.onWorkerStop != nil {
			p.onWorkerStop(worker)
		}
	}
}

// restartWorker restarts a worker
func (p *WorkerPool) restartWorker(worker *Worker) {
	p.removeWorker(worker.ID)
	p.addWorker()
}

// getIdleWorker returns an idle worker
func (p *WorkerPool) getIdleWorker() *Worker {
	p.workersMu.RLock()
	defer p.workersMu.RUnlock()

	for _, worker := range p.workers {
		if worker.state.Load().(WorkerState) == WorkerStateIdle {
			return worker
		}
	}
	return nil
}

// run is the main worker loop
func (w *Worker) run() {
	defer w.pool.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			w.state.Store(WorkerStateStopped)
			return
		case job := <-w.pool.jobChan:
			w.process(job)
		}
	}
}

// process processes a single job
func (w *Worker) process(job *EmailJob) {
	w.state.Store(WorkerStateProcessing)
	w.lastJob.Store(job)
	defer func() {
		w.state.Store(WorkerStateIdle)
	}()

	if w.pool.onJobStart != nil {
		w.pool.onJobStart(job)
	}

	start := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(w.ctx, w.pool.config.ProcessTimeout)
	defer cancel()

	// Send email
	err := w.pool.sender.Send(ctx, job)
	duration := time.Since(start)

	result := &JobResult{
		JobID:      job.ID,
		Success:    err == nil,
		Error:      err,
		Duration:   duration,
		WorkerID:   w.ID,
		RetryCount: job.Attempts,
	}

	if err == nil {
		w.processed.Add(1)
	} else {
		w.failed.Add(1)
	}

	// Send result
	select {
	case w.pool.resultChan <- result:
	default:
		log.Printf("Result channel full, job %s result dropped", job.ID)
	}

	if w.pool.onJobComplete != nil {
		w.pool.onJobComplete(job, result)
	}
}

// SetCallbacks sets worker pool callbacks
func (p *WorkerPool) SetCallbacks(
	onJobStart func(*EmailJob),
	onJobComplete func(*EmailJob, *JobResult),
	onJobFail func(*EmailJob, error),
	onWorkerStart func(*Worker),
	onWorkerStop func(*Worker),
) {
	p.onJobStart = onJobStart
	p.onJobComplete = onJobComplete
	p.onJobFail = onJobFail
	p.onWorkerStart = onWorkerStart
	p.onWorkerStop = onWorkerStop
}

// WorkerCount returns the current number of workers
func (p *WorkerPool) WorkerCount() int {
	p.workersMu.RLock()
	defer p.workersMu.RUnlock()
	return len(p.workers)
}

// GetWorkerStats returns statistics for all workers
func (p *WorkerPool) GetWorkerStats() []WorkerStats {
	p.workersMu.RLock()
	defer p.workersMu.RUnlock()

	stats := make([]WorkerStats, 0, len(p.workers))
	for _, worker := range p.workers {
		stats = append(stats, WorkerStats{
			ID:        worker.ID,
			State:     worker.state.Load().(WorkerState),
			Started:   worker.started,
			Processed: worker.processed.Load(),
			Failed:    worker.failed.Load(),
		})
	}
	return stats
}

// WorkerStats holds statistics for a single worker
type WorkerStats struct {
	ID        string      `json:"id"`
	State     WorkerState `json:"state"`
	Started   time.Time   `json:"started"`
	Processed int64       `json:"processed"`
	Failed    int64       `json:"failed"`
}

// Metrics recording
func (m *WorkerMetrics) recordJobDispatched() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Track for rate calculation
}

func (m *WorkerMetrics) recordJobCompleted(result *JobResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.JobsProcessed++
	if result.Success {
		m.JobsSucceeded++
	} else {
		m.JobsFailed++
	}

	// Record processing time
	if len(m.ProcessingTimes) >= m.maxSamples {
		m.ProcessingTimes = m.ProcessingTimes[1:]
	}
	m.ProcessingTimes = append(m.ProcessingTimes, result.Duration)
}

// GetMetrics returns current metrics
func (p *WorkerPool) GetMetrics() WorkerPoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	avgTime := time.Duration(0)
	if len(p.metrics.ProcessingTimes) > 0 {
		var total time.Duration
		for _, t := range p.metrics.ProcessingTimes {
			total += t
		}
		avgTime = total / time.Duration(len(p.metrics.ProcessingTimes))
	}

	return WorkerPoolMetrics{
		ActiveWorkers:     int(p.metrics.ActiveWorkers),
		JobsProcessed:     p.metrics.JobsProcessed,
		JobsSucceeded:     p.metrics.JobsSucceeded,
		JobsFailed:        p.metrics.JobsFailed,
		QueueDepth:        p.metrics.QueueDepth,
		AvgProcessingTime: avgTime,
	}
}

// WorkerPoolMetrics holds aggregated metrics
type WorkerPoolMetrics struct {
	ActiveWorkers     int           `json:"active_workers"`
	JobsProcessed     int64         `json:"jobs_processed"`
	JobsSucceeded     int64         `json:"jobs_succeeded"`
	JobsFailed        int64         `json:"jobs_failed"`
	QueueDepth        int64         `json:"queue_depth"`
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return &RateLimiter{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens = math.Min(r.maxTokens, r.tokens+elapsed*r.refillRate)
	r.lastRefill = now

	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}

// Wait waits until a request is allowed or context is cancelled
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
