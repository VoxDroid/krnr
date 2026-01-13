package adapters

import (
	"context"
	"testing"
)

type fakeRegistry struct{ items []CommandSetSummary }

func (f *fakeRegistry) ListCommandSets(ctx context.Context) ([]CommandSetSummary, error) {
	return f.items, nil
}
func (f *fakeRegistry) GetCommandSet(ctx context.Context, name string) (CommandSetSummary, error) {
	for _, it := range f.items {
		if it.Name == name {
			return it, nil
		}
	}
	return CommandSetSummary{}, ErrNotFound
}
func (f *fakeRegistry) SaveCommandSet(ctx context.Context, cs CommandSetSummary) error { return nil }
func (f *fakeRegistry) DeleteCommandSet(ctx context.Context, name string) error        { return nil }

type fakeExecutor struct{}

type localFakeRunHandle struct{ ch chan RunEvent }

func (l *localFakeRunHandle) Events() <-chan RunEvent { return l.ch }
func (l *localFakeRunHandle) Cancel()                 {}

func (f *fakeExecutor) Run(ctx context.Context, name string, args []string) (RunHandle, error) {
	return &localFakeRunHandle{ch: make(chan RunEvent)}, nil
}

func TestFakeAdapters_List(t *testing.T) {
	reg := &fakeRegistry{items: []CommandSetSummary{{Name: "a", Description: "A"}, {Name: "b", Description: "B"}}}
	items, err := reg.ListCommandSets(context.Background())
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items got %d", len(items))
	}
}
