// Package ui contains tests for the TUI package.
package ui

import (
	"context"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
)

func TestSaveViaModel(t *testing.T) {
	// fake registry that records saves
	f := &saveFakeRegistry{}
	ui := modelpkg.New(f, nil, nil, nil)
	cs := adapters.CommandSetSummary{Name: "nset", Description: "desc", Commands: []string{"echo hi"}, AuthorName: "me"}
	if err := ui.Save(context.Background(), cs); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if f.last.Name != "nset" {
		t.Fatalf("expected saved name nset got %s", f.last.Name)
	}
}

type saveFakeRegistry struct{ last adapters.CommandSetSummary }

func (s *saveFakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) {
	return nil, nil
}
func (s *saveFakeRegistry) GetCommandSet(_ context.Context, _ string) (adapters.CommandSetSummary, error) {
	return adapters.CommandSetSummary{}, adapters.ErrNotFound
}
func (s *saveFakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) {
	return nil, adapters.ErrNotFound
}
func (s *saveFakeRegistry) SaveCommandSet(_ context.Context, cs adapters.CommandSetSummary) error {
	s.last = cs
	return nil
}
func (s *saveFakeRegistry) DeleteCommandSet(_ context.Context, _ string) error { return nil }
func (s *saveFakeRegistry) ReplaceCommands(_ context.Context, _ string, _ []string) error {
	return nil
}
func (s *saveFakeRegistry) UpdateCommandSet(_ context.Context, _ string, cs adapters.CommandSetSummary) error {
	s.last = cs
	return nil
}
func (s *saveFakeRegistry) ListVersionsByName(_ context.Context, _ string) ([]adapters.Version, error) {
	return nil, nil
}
func (s *saveFakeRegistry) ApplyVersionByName(_ context.Context, _ string, _ int) error { return nil }
