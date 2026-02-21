package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/soochol/upal/internal/upal"
)

const maxRunRecords = 1000

// MemoryRunRepository stores run records in memory with FIFO eviction.
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

	// FIFO eviction when at capacity.
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

	// Sort by created_at descending (newest first).
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	total := len(filtered)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
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

	// Sort by created_at descending.
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}
