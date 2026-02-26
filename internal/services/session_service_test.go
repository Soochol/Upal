package services_test

import (
	"context"
	"strings"
	"testing"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSessionSvc() *services.SessionService {
	return services.NewSessionService(repository.NewMemorySessionRepository())
}

func TestSessionService_Create(t *testing.T) {
	svc := newTestSessionSvc()
	sess, err := svc.Create(context.Background(), &upal.Session{Name: "Test"})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(sess.ID, "sess-"))
	assert.Equal(t, upal.SessionStatusDraft, sess.Status)
	assert.False(t, sess.CreatedAt.IsZero())
	assert.False(t, sess.UpdatedAt.IsZero())
}

func TestSessionService_Create_NameRequired(t *testing.T) {
	svc := newTestSessionSvc()
	_, err := svc.Create(context.Background(), &upal.Session{})
	assert.Error(t, err)
}

func TestSessionService_Create_GeneratesStageIDs(t *testing.T) {
	svc := newTestSessionSvc()
	sess, err := svc.Create(context.Background(), &upal.Session{
		Name:   "Test",
		Stages: []upal.Stage{{Name: "Collect", Type: "collect"}},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, sess.Stages[0].ID)
	assert.True(t, strings.HasPrefix(sess.Stages[0].ID, "stg-"))
}

func TestSessionService_Create_PreservesExistingStageIDs(t *testing.T) {
	svc := newTestSessionSvc()
	sess, err := svc.Create(context.Background(), &upal.Session{
		Name:   "Test",
		Stages: []upal.Stage{{ID: "stg-existing", Name: "Collect", Type: "collect"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "stg-existing", sess.Stages[0].ID)
}

func TestSessionService_CRUD(t *testing.T) {
	svc := newTestSessionSvc()
	ctx := context.Background()

	// Create
	sess, err := svc.Create(ctx, &upal.Session{Name: "Original"})
	require.NoError(t, err)

	// Get
	got, err := svc.Get(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "Original", got.Name)

	// List
	list, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// Update
	got.Name = "Updated"
	err = svc.Update(ctx, got)
	require.NoError(t, err)

	got2, err := svc.Get(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got2.Name)
	assert.True(t, got2.UpdatedAt.After(got2.CreatedAt) || got2.UpdatedAt.Equal(got2.CreatedAt))

	// Delete
	err = svc.Delete(ctx, sess.ID)
	require.NoError(t, err)

	_, err = svc.Get(ctx, sess.ID)
	assert.Error(t, err)
}

func TestSessionService_UpdateSettings(t *testing.T) {
	svc := newTestSessionSvc()
	ctx := context.Background()

	sess, err := svc.Create(ctx, &upal.Session{Name: "Test"})
	require.NoError(t, err)

	err = svc.UpdateSettings(ctx, sess.ID, upal.SessionSettings{
		Name:  "Renamed",
		Model: "anthropic/claude-sonnet-4-20250514",
	})
	require.NoError(t, err)

	got, err := svc.Get(ctx, sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Name)
	assert.Equal(t, "anthropic/claude-sonnet-4-20250514", got.Model)
}

func TestSessionService_UpdateSettings_RejectsArchived(t *testing.T) {
	svc := newTestSessionSvc()
	ctx := context.Background()

	sess, err := svc.Create(ctx, &upal.Session{Name: "Test"})
	require.NoError(t, err)

	// Set status to archived
	sess.Status = upal.SessionStatusArchived
	err = svc.Update(ctx, sess)
	require.NoError(t, err)

	err = svc.UpdateSettings(ctx, sess.ID, upal.SessionSettings{Name: "New"})
	assert.Error(t, err)
}

func TestSessionService_UpdateSettings_ClearSchedule(t *testing.T) {
	svc := newTestSessionSvc()
	ctx := context.Background()

	sess, err := svc.Create(ctx, &upal.Session{Name: "Test", Schedule: "0 * * * *"})
	require.NoError(t, err)

	err = svc.UpdateSettings(ctx, sess.ID, upal.SessionSettings{ClearSchedule: true})
	require.NoError(t, err)

	got, err := svc.Get(ctx, sess.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Schedule)
}
