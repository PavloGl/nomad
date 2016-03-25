package consul

import (
	"log"
	"math/rand"
	"sync"
	"time"

	cstructs "github.com/hashicorp/nomad/client/driver/structs"
)

// CheckRunner runs a given check in a specific interval and update a
// corresponding Consul TTL check
type CheckRunner struct {
	check    Check
	runCheck func(Check)
	logger   *log.Logger
	stop     bool
	stopCh   chan struct{}
	stopLock sync.Mutex

	started     bool
	startedLock sync.Mutex
}

// NewCheckRunner configures and returns a CheckRunner
func NewCheckRunner(check Check, runCheck func(Check), logger *log.Logger) *CheckRunner {
	nc := NomadCheck{
		check:    check,
		runCheck: runCheck,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
	return &nc
}

// Start is used to start the check. The check runs until stop is called
func (r *CheckRunner) Start() {
	r.startedLock.Lock()
	if r.started {
		return
	}
	r.started = true
	r.stopLock.Lock()
	defer r.stopLock.Unlock()
	r.stopCh = make(chan struct{})
	go r.run()
}

// Stop is used to stop the check.
func (r *CheckRunner) Stop() {
	r.stopLock.Lock()
	defer r.stopLock.Unlock()
	if !r.stop {
		r.stop = true
		close(r.stopCh)
	}
}

// run is invoked by a goroutine to run until Stop() is called
func (r *CheckRunner) run() {
	// Get the randomized initial pause time
	initialPauseTime := randomStagger(r.check.Interval())
	r.logger.Printf("[DEBUG] agent: pausing %v before first invocation of %s", initialPauseTime, r.check.ID())
	next := time.After(initialPauseTime)
	for {
		select {
		case <-next:
			r.runCheck(r.check)
			next = time.After(r.check.Interval())
		case <-r.stopCh:
			return
		}
	}
}

// Check is an interface which check providers can implement for Nomad to run
type Check interface {
	Run() *cstructs.CheckResult
	ID() string
	Interval() time.Duration
}

// Returns a random stagger interval between 0 and the duration
func randomStagger(intv time.Duration) time.Duration {
	return time.Duration(uint64(rand.Int63()) % uint64(intv))
}
