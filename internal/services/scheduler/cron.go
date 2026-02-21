package scheduler

// cron.go — CronManager responsibility layer.
// Cron expression parsing and cron entry registration/removal,
// implemented as methods on SchedulerService (same package, no circular import).

import (
	"log/slog"

	"github.com/robfig/cron/v3"
	"github.com/soochol/upal/internal/upal"
)

// parseCronExpr tries 6-field (with seconds) then 5-field (standard) parsing.
// If timezone is non-empty and non-UTC, it is applied via the CRON_TZ= prefix.
func parseCronExpr(expr string, timezone string) (cron.Schedule, error) {
	if timezone != "" && timezone != "UTC" {
		expr = "CRON_TZ=" + timezone + " " + expr
	}
	parser6 := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser6.Parse(expr)
	if err == nil {
		return sched, nil
	}
	parser5 := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return parser5.Parse(expr)
}

// registerCronJob parses the schedule's cron expression, registers a new cron
// entry, and stores the resulting EntryID in entryMap.
func (s *SchedulerService) registerCronJob(schedule *upal.Schedule) error {
	cronSched, err := parseCronExpr(schedule.CronExpr, schedule.Timezone)
	if err != nil {
		return err
	}

	entryID := s.cron.Schedule(cronSched, cron.FuncJob(func() {
		s.executeScheduledRun(schedule)
	}))

	s.mu.Lock()
	s.entryMap[schedule.ID] = entryID
	s.mu.Unlock()

	if schedule.PipelineID != "" {
		slog.Info("scheduler: registered cron job",
			"id", schedule.ID, "pipeline", schedule.PipelineID, "cron", schedule.CronExpr)
	} else {
		slog.Info("scheduler: registered cron job",
			"id", schedule.ID, "workflow", schedule.WorkflowName, "cron", schedule.CronExpr)
	}
	return nil
}
