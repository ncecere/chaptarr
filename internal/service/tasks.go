package service

import (
	"context"
	"sync"
	"time"

	"github.com/ncecere/chaptarr/internal/model"
)

type taskDef struct {
	name, label string
	// interval reads the current schedule; 0 means on demand only.
	interval func(model.Settings) time.Duration
	// manual is true when the user started the task.
	run func(ctx context.Context, manual bool) error
}

type taskState struct {
	running   bool
	lastRun   *time.Time
	lastDur   time.Duration
	lastError string
	// due is when the scheduler should next start it.
	due time.Time
}

type taskRunner struct {
	svc   *Service
	defs  []taskDef
	mu    sync.Mutex
	state map[string]*taskState
}

func newTaskRunner(s *Service) *taskRunner {
	r := &taskRunner{svc: s, state: map[string]*taskState{}}
	r.defs = []taskDef{
		{
			name: "syncQueue", label: "Check downloads and import",
			interval: func(model.Settings) time.Duration { return 15 * time.Second },
			run:      func(ctx context.Context, _ bool) error { return s.SyncQueue(ctx) },
		},
		{
			name: "searchMissing", label: "Search for missing books",
			interval: func(set model.Settings) time.Duration {
				return time.Duration(set.MissingSearchIntervalMinutes) * time.Minute
			},
			run: func(ctx context.Context, manual bool) error {
				if manual {
					return s.searchMissing(ctx, 0, 0, 0)
				}
				// Scheduled runs work through the backlog a slice at a time
				// and leave a book alone for a day after searching for it.
				return s.searchMissing(ctx, 0, 24*time.Hour, 25)
			},
		},
		{
			name: "refreshAuthors", label: "Refresh monitored authors",
			interval: func(set model.Settings) time.Duration {
				return time.Duration(set.AuthorRefreshIntervalHours) * time.Hour
			},
			run: func(ctx context.Context, _ bool) error { return s.refreshAuthors(ctx) },
		},
		{
			name: "scanLibrary", label: "Scan library folders",
			interval: func(model.Settings) time.Duration { return 12 * time.Hour },
			run:      func(ctx context.Context, _ bool) error { return s.ScanLibrary(ctx) },
		},
	}
	// Stagger the first runs so start-up does not hit the metadata provider
	// and the indexers at once.
	now := time.Now()
	first := map[string]time.Duration{
		"syncQueue": 5 * time.Second, "scanLibrary": time.Minute,
		"refreshAuthors": 10 * time.Minute, "searchMissing": 15 * time.Minute,
	}
	for _, d := range r.defs {
		r.state[d.name] = &taskState{due: now.Add(first[d.name])}
	}
	return r
}

// Run drives the schedule until ctx is cancelled.
func (s *Service) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			set, err := s.Store.Settings()
			if err != nil {
				continue
			}
			for _, d := range s.tasks.defs {
				if d.interval(set) <= 0 {
					continue
				}
				s.tasks.mu.Lock()
				due := !s.tasks.state[d.name].due.After(now)
				s.tasks.mu.Unlock()
				if due {
					s.tasks.start(ctx, d, false)
				}
			}
		}
	}
}

// start runs the task in the background unless it is already running.
func (r *taskRunner) start(ctx context.Context, d taskDef, manual bool) bool {
	r.mu.Lock()
	st := r.state[d.name]
	if st.running {
		r.mu.Unlock()
		return false
	}
	st.running = true
	r.mu.Unlock()

	go func() {
		began := time.Now()
		var err error
		func() {
			defer func() {
				if p := recover(); p != nil {
					r.svc.Log.Error("task panicked", "task", d.name, "panic", p)
					err = &Error{500, "internal error"}
				}
			}()
			err = d.run(ctx, manual)
		}()
		set, _ := r.svc.Store.Settings()
		r.mu.Lock()
		st.running = false
		st.lastRun = &began
		st.lastDur = time.Since(began)
		st.lastError = ""
		if err != nil && ctx.Err() == nil {
			st.lastError = err.Error()
			r.svc.Log.Warn("task failed", "task", d.name, "error", err)
		}
		if iv := d.interval(set); iv > 0 {
			st.due = time.Now().Add(iv)
		}
		r.mu.Unlock()
	}()
	return true
}

// RunTask starts a task on demand.
func (s *Service) RunTask(name string) error {
	for _, d := range s.tasks.defs {
		if d.name == name {
			if !s.tasks.start(context.Background(), d, true) {
				return conflict("That task is already running")
			}
			return nil
		}
	}
	return ErrNotFound
}

func (s *Service) Tasks() ([]model.Task, error) {
	set, err := s.Store.Settings()
	if err != nil {
		return nil, err
	}
	s.tasks.mu.Lock()
	defer s.tasks.mu.Unlock()
	out := make([]model.Task, 0, len(s.tasks.defs))
	for _, d := range s.tasks.defs {
		st := s.tasks.state[d.name]
		t := model.Task{
			Name: d.name, Label: d.label, IntervalSeconds: int(d.interval(set).Seconds()),
			Running: st.running, LastDurationMs: st.lastDur.Milliseconds(), LastError: st.lastError,
		}
		if st.lastRun != nil {
			u := st.lastRun.UTC()
			t.LastRunAt = &u
		}
		if t.IntervalSeconds > 0 {
			due := st.due.UTC()
			t.NextRunAt = &due
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *Service) refreshAuthors(ctx context.Context) error {
	authors, err := s.Store.Authors()
	if err != nil {
		return err
	}
	var firstErr error
	for i := range authors {
		a := &authors[i]
		if !a.Monitored {
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		monitorNew := a.MonitorNewBooks
		if err := s.syncAuthor(ctx, a, func(model.BookResult) bool { return monitorNew }); err != nil {
			s.Log.Warn("author refresh failed", "author", a.Name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return firstErr
}
