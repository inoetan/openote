package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"

	"github.com/inoetan/openote/api/pkg/models"
)

const (
	leaderLockTTL    = 30 * time.Second
	leaderRenewEvery = 10 * time.Second
)

// Scheduler wraps robfig/cron and manages distributed schedule execution.
// Only the instance holding the Redis leader lock will fire jobs.
type Scheduler struct {
	cron     *cron.Cron
	repo     Repository
	redis    *redis.Client
	mu       sync.Mutex
	entryIDs map[string]cron.EntryID // scheduleID (string) → cron entry ID
}

// NewScheduler creates a Scheduler but does not start it.
func NewScheduler(repo Repository, redisClient *redis.Client) *Scheduler {
	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC))
	return &Scheduler{
		cron:     c,
		repo:     repo,
		redis:    redisClient,
		entryIDs: make(map[string]cron.EntryID),
	}
}

// Start loads all active schedules from the database, registers them with the
// cron runner, and begins the leader-lock renewal loop.
func (s *Scheduler) Start(ctx context.Context) error {
	schedules, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("load schedules: %w", err)
	}

	for _, sc := range schedules {
		if err := s.RegisterJob(ctx, sc); err != nil {
			log.Printf("scheduler: failed to register schedule %s: %v", sc.ID, err)
		}
	}

	s.cron.Start()

	// Background goroutine: attempt to acquire/renew leader lock.
	go s.runLeaderLoop(ctx)

	return nil
}

// Stop halts the cron scheduler gracefully.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// RegisterJob adds or replaces a cron entry for the given schedule.
func (s *Scheduler) RegisterJob(ctx context.Context, sc *Schedule) error {
	_, err := time.LoadLocation(sc.Timezone)
	if err != nil {
		sc.Timezone = "UTC"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove old entry if present.
	if eid, ok := s.entryIDs[sc.ID.String()]; ok {
		s.cron.Remove(eid)
	}

	scheduleID := sc.ID
	jobID := sc.JobID

	cronSched, err := parseCronExpr(sc.CronExpr)
	if err != nil {
		return fmt.Errorf("parse cron expr %q: %w", sc.CronExpr, err)
	}

	eid := s.cron.Schedule(cronSched, cron.FuncJob(func() {
		s.fireJob(context.Background(), scheduleID, jobID)
	}))

	s.entryIDs[sc.ID.String()] = eid
	return nil
}

// UnregisterJob removes the cron entry for the given scheduleID.
func (s *Scheduler) UnregisterJob(scheduleID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if eid, ok := s.entryIDs[scheduleID]; ok {
		s.cron.Remove(eid)
		delete(s.entryIDs, scheduleID)
	}
}

// fireJob checks the distributed leader lock and, if held, pushes a dispatch
// message to the Redis Stream and updates schedule timing.
func (s *Scheduler) fireJob(ctx context.Context, scheduleID, jobID uuid.UUID) {
	// Only the leader fires jobs.
	if !s.isLeader(ctx) {
		return
	}

	dispatch := models.JobDispatch{
		JobID: jobID.String(),
	}
	payload, err := json.Marshal(dispatch)
	if err != nil {
		log.Printf("scheduler: marshal dispatch: %v", err)
		return
	}

	err = s.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: models.RedisStreamKey,
		Values: map[string]interface{}{
			"payload": string(payload),
		},
	}).Err()
	if err != nil {
		log.Printf("scheduler: xadd execution: %v", err)
		return
	}

	// Update last/next run times.
	now := time.Now().UTC()
	sc, err := s.repo.FindByID(ctx, scheduleID)
	if err != nil || sc == nil {
		return
	}

	cronParsed, err := parseCronExpr(sc.CronExpr)
	if err != nil {
		log.Printf("scheduler: re-parse cron %q: %v", sc.CronExpr, err)
		return
	}
	nextRun := cronParsed.Next(now)

	if err := s.repo.UpdateLastRun(ctx, scheduleID, now, nextRun); err != nil {
		log.Printf("scheduler: update last_run for %s: %v", scheduleID, err)
	}
}

// isLeader attempts to acquire (or verify) the distributed Redis lock.
func (s *Scheduler) isLeader(ctx context.Context) bool {
	ok, err := s.redis.SetNX(ctx, models.RedisSchedulerLock, "1", leaderLockTTL).Result()
	if err != nil {
		return false
	}
	return ok
}

// runLeaderLoop periodically renews the leader lock so it doesn't expire
// while this instance is active.
func (s *Scheduler) runLeaderLoop(ctx context.Context) {
	ticker := time.NewTicker(leaderRenewEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Extend TTL if we already hold the lock.
			_ = s.redis.Expire(ctx, models.RedisSchedulerLock, leaderLockTTL)
		}
	}
}

// parseCronExpr tries 6-field (with seconds) then standard 5-field.
func parseCronExpr(expr string) (cron.Schedule, error) {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(expr)
	if err == nil {
		return sched, nil
	}
	return cron.ParseStandard(expr)
}
