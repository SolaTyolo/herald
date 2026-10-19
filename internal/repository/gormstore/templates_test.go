package gormstore_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/repository/gormstore"
)

func TestTemplateRoundTripSQLite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := gormstore.Open(ctx, config.DBDriverSQLite, filepath.Join(t.TempDir(), "herald.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.RunMigrations(ctx); err != nil {
		t.Fatal(err)
	}

	tmpl := &domain.NotificationTemplate{Name: "welcome", Generation: domain.TemplateGenerationDynamic}
	if err := store.CreateTemplate(ctx, "env-1", tmpl); err != nil {
		t.Fatal(err)
	}
	ver := &domain.TemplateVersion{Name: "v1", Subject: "Hi {{.name}}", HTMLContent: "<b>{{.name}}</b>", Active: true}
	if err := store.CreateTemplateVersion(ctx, "env-1", tmpl.ID, ver); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetTemplate(ctx, "env-1", tmpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ActiveVersion() == nil || got.ActiveVersion().Subject != "Hi {{.name}}" {
		t.Fatalf("got %+v", got)
	}

	if err := store.DeleteTemplate(ctx, "env-1", tmpl.ID); err != nil {
		t.Fatal(err)
	}
	_, err = store.GetTemplate(ctx, "env-1", tmpl.ID)
	if !repository.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
