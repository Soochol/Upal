package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/soochol/upal/internal/upal"
)

const maxRunRecords = 1000

type MemoryRunRepository struct {
	mu      sync.RWMutex
	records map[string]*upal.RunRecord
	order   []string // insertion order for FIFO eviction
}

func NewMemoryRunRepository() *MemoryRunRepository {
	return &MemoryRunRepository{
		records: make(map[string]*upal.RunRecord),
	}
}

func (r *MemoryRunRepository) Create(_ context.Context, record *upal.RunRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.order) >= maxRunRecords {
		oldest := r.order[0]
		r.order = r.order[1:]
		delete(r.records, oldest)
	}

	r.records[record.ID] = record
	r.order = append(r.order, record.ID)
	return nil
}

func (r *MemoryRunRepository) Get(_ context.Context, id string) (*upal.RunRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rec, ok := r.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return rec, nil
}

func (r *MemoryRunRepository) Update(_ context.Context, record *upal.RunRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.records[record.ID]; !ok {
		return ErrNotFound
	}
	r.records[record.ID] = record
	return nil
}

func (r *MemoryRunRepository) ListByWorkflow(_ context.Context, workflowName string, limit, offset int) ([]*upal.RunRecord, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*upal.RunRecord
	for _, rec := range r.records {
		if rec.WorkflowName == workflowName {
			filtered = append(filtered, rec)
		}
	}

	return sortAndPaginate(filtered, limit, offset), len(filtered), nil
}

func (r *MemoryRunRepository) ListAll(_ context.Context, limit, offset int, status string) ([]*upal.RunRecord, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make([]*upal.RunRecord, 0, len(r.records))
	for _, rec := range r.records {
		if status == "" || string(rec.Status) == status {
			all = append(all, rec)
		}
	}

	return sortAndPaginate(all, limit, offset), len(all), nil
}

// sortAndPaginate sorts runs by CreatedAt descending and returns the requested page.
func sortAndPaginate(runs []*upal.RunRecord, limit, offset int) []*upal.RunRecord {
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})

	if offset >= len(runs) {
		return nil
	}
	end := offset + limit
	if end > len(runs) {
		end = len(runs)
	}
	return runs[offset:end]
}
