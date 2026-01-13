package adapters

import (
	"context"
	"fmt"

	"github.com/VoxDroid/krnr/internal/registry"
)

// RegistryAdapterImpl adapts internal/registry.Repository to the UI adapters.RegistryAdapter interface.
type RegistryAdapterImpl struct{ repo *registry.Repository }

func NewRegistryAdapter(repo *registry.Repository) *RegistryAdapterImpl { return &RegistryAdapterImpl{repo: repo} }

func (r *RegistryAdapterImpl) ListCommandSets(ctx context.Context) ([]CommandSetSummary, error) {
	sets, err := r.repo.ListCommandSets()
	if err != nil { return nil, fmt.Errorf("list command sets: %w", err) }
	out := make([]CommandSetSummary, 0, len(sets))
	for _, s := range sets {
		out = append(out, CommandSetSummary{Name: s.Name, Description: s.Description.String})
	}
	return out, nil
}

func (r *RegistryAdapterImpl) GetCommandSet(ctx context.Context, name string) (CommandSetSummary, error) {
	s, err := r.repo.GetCommandSetByName(name)
	if err != nil { return CommandSetSummary{}, fmt.Errorf("get: %w", err) }
	if s == nil { return CommandSetSummary{}, ErrNotFound }
	// map repository CommandSet into the adapter summary
	cmds := make([]string, 0, len(s.Commands))
	for _, c := range s.Commands { cmds = append(cmds, c.Command) }
	return CommandSetSummary{
		Name: s.Name,
		Description: s.Description.String,
		Commands: cmds,
		AuthorName: s.AuthorName.String,
		AuthorEmail: s.AuthorEmail.String,
		Tags: s.Tags,
		CreatedAt: s.CreatedAt,
		LastRun: s.LastRun.String,
	}, nil
}

func (r *RegistryAdapterImpl) GetCommands(ctx context.Context, name string) ([]string, error) {
	s, err := r.repo.GetCommandSetByName(name)
	if err != nil { return nil, fmt.Errorf("get commands: %w", err) }
	if s == nil { return nil, ErrNotFound }
	out := make([]string, 0, len(s.Commands))
	for _, c := range s.Commands { out = append(out, c.Command) }
	return out, nil
}

func (r *RegistryAdapterImpl) SaveCommandSet(ctx context.Context, cs CommandSetSummary) error {
	// Map CommandSetSummary (with Commands) into repository CreateCommandSet
	var desc *string
	if cs.Description != "" { desc = &cs.Description }
	var an *string
	if cs.AuthorName != "" { an = &cs.AuthorName }
	var ae *string
	if cs.AuthorEmail != "" { ae = &cs.AuthorEmail }
	_, err := r.repo.CreateCommandSet(cs.Name, desc, an, ae, cs.Commands)
	return err
}

func (r *RegistryAdapterImpl) DeleteCommandSet(ctx context.Context, name string) error { return r.repo.DeleteCommandSet(name) }

func (r *RegistryAdapterImpl) ReplaceCommands(ctx context.Context, name string, commands []string) error {
	cs, err := r.repo.GetCommandSetByName(name)
	if err != nil { return err }
	if cs == nil { return ErrNotFound }
	return r.repo.ReplaceCommands(cs.ID, commands)
}
