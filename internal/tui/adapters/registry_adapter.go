package adapters

import (
	"context"
	"fmt"

	"github.com/VoxDroid/krnr/internal/registry"
)

// RegistryAdapterImpl adapts internal/registry.Repository to the UI adapters.RegistryAdapter interface.
type RegistryAdapterImpl struct{ repo *registry.Repository }

// NewRegistryAdapter returns an adapter that wraps an internal registry.Repository.
func NewRegistryAdapter(repo *registry.Repository) *RegistryAdapterImpl {
	return &RegistryAdapterImpl{repo: repo}
}

// ListCommandSets returns a list of command set summaries.
func (r *RegistryAdapterImpl) ListCommandSets(_ context.Context) ([]CommandSetSummary, error) {
	sets, err := r.repo.ListCommandSets()
	if err != nil {
		return nil, fmt.Errorf("list command sets: %w", err)
	}
	out := make([]CommandSetSummary, 0, len(sets))
	for _, s := range sets {
		out = append(out, CommandSetSummary{Name: s.Name, Description: s.Description.String})
	}
	return out, nil
}

// GetCommandSet retrieves a full CommandSetSummary by name.
func (r *RegistryAdapterImpl) GetCommandSet(_ context.Context, name string) (CommandSetSummary, error) {
	s, err := r.repo.GetCommandSetByName(name)
	if err != nil {
		return CommandSetSummary{}, fmt.Errorf("get: %w", err)
	}
	if s == nil {
		return CommandSetSummary{}, ErrNotFound
	}
	// map repository CommandSet into the adapter summary
	cmds := make([]string, 0, len(s.Commands))
	for _, c := range s.Commands {
		cmds = append(cmds, c.Command)
	}
	return CommandSetSummary{
		Name:        s.Name,
		Description: s.Description.String,
		Commands:    cmds,
		AuthorName:  s.AuthorName.String,
		AuthorEmail: s.AuthorEmail.String,
		Tags:        s.Tags,
		CreatedAt:   s.CreatedAt,
		LastRun:     s.LastRun.String,
	}, nil
}

// GetCommands returns only the commands for a named command set.
func (r *RegistryAdapterImpl) GetCommands(_ context.Context, name string) ([]string, error) {
	s, err := r.repo.GetCommandSetByName(name)
	if err != nil {
		return nil, fmt.Errorf("get commands: %w", err)
	}
	if s == nil {
		return nil, ErrNotFound
	}
	out := make([]string, 0, len(s.Commands))
	for _, c := range s.Commands {
		out = append(out, c.Command)
	}
	return out, nil
}

// SaveCommandSet creates a new command set in the underlying repository.
func (r *RegistryAdapterImpl) SaveCommandSet(_ context.Context, cs CommandSetSummary) error {
	// Map CommandSetSummary (with Commands) into repository CreateCommandSet
	var desc *string
	if cs.Description != "" {
		desc = &cs.Description
	}
	var an *string
	if cs.AuthorName != "" {
		an = &cs.AuthorName
	}
	var ae *string
	if cs.AuthorEmail != "" {
		ae = &cs.AuthorEmail
	}
	_, err := r.repo.CreateCommandSet(cs.Name, desc, an, ae, cs.Commands)
	return err
}

// DeleteCommandSet deletes a command set by name.
func (r *RegistryAdapterImpl) DeleteCommandSet(_ context.Context, name string) error {
	return r.repo.DeleteCommandSet(name)
}

// ReplaceCommands replaces the commands for the named set.
func (r *RegistryAdapterImpl) ReplaceCommands(_ context.Context, name string, commands []string) error {
	cs, err := r.repo.GetCommandSetByName(name)
	if err != nil {
		return err
	}
	if cs == nil {
		return ErrNotFound
	}
	return r.repo.ReplaceCommands(cs.ID, commands)
}

// UpdateCommandSet updates metadata and tags for an existing set.
func (r *RegistryAdapterImpl) UpdateCommandSet(_ context.Context, oldName string, cs CommandSetSummary) error {
	cur, err := r.repo.GetCommandSetByName(oldName)
	if err != nil {
		return err
	}
	if cur == nil {
		return ErrNotFound
	}
	// prepare pointers for nullable fields
	var desc *string
	if cs.Description != "" {
		desc = &cs.Description
	}
	var an *string
	if cs.AuthorName != "" {
		an = &cs.AuthorName
	}
	var ae *string
	if cs.AuthorEmail != "" {
		ae = &cs.AuthorEmail
	}
	return r.repo.UpdateCommandSet(cur.ID, cs.Name, desc, an, ae, cs.Tags)
}

// ListVersionsByName lists historical versions for the named command set.
func (r *RegistryAdapterImpl) ListVersionsByName(_ context.Context, name string) ([]Version, error) {
	vers, err := r.repo.ListVersionsByName(name)
	if err != nil {
		return nil, err
	}
	out := make([]Version, 0, len(vers))
	for _, v := range vers {
		out = append(out, Version{
			Version:     v.Version,
			CreatedAt:   v.CreatedAt,
			AuthorName:  v.AuthorName.String,
			AuthorEmail: v.AuthorEmail.String,
			Description: v.Description.String,
			Commands:    v.Commands,
			Operation:   v.Operation,
		})
	}
	return out, nil
}

// ApplyVersionByName applies a historical version to the named set (rollback).
func (r *RegistryAdapterImpl) ApplyVersionByName(_ context.Context, name string, versionNum int) error {
	return r.repo.ApplyVersionByName(name, versionNum)
}
