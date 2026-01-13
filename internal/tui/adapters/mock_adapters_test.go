package adapters

import (
	"context"
	"testing"
)

type fakeRegistry struct{ items []CommandSetSummary }

func (f *fakeRegistry) ListCommandSets(_ context.Context) ([]CommandSetSummary, error) {
	return f.items, nil
}
func (f *fakeRegistry) GetCommandSet(_ context.Context, name string) (CommandSetSummary, error) {
	for _, it := range f.items {
		if it.Name == name {
			return it, nil
		}
	}
	return CommandSetSummary{}, ErrNotFound
}
func (f *fakeRegistry) SaveCommandSet(_ context.Context, _ CommandSetSummary) error { return nil }
func (f *fakeRegistry) DeleteCommandSet(_ context.Context, _ string) error        { return nil }
func (f *fakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) {
	return []string{}, nil
}
func (f *fakeRegistry) ReplaceCommands(_ context.Context, _ string, _ []string) error { return nil }
func (f *fakeRegistry) UpdateCommandSet(_ context.Context, _ string, _ CommandSetSummary) error {
	return nil
}
func (f *fakeRegistry) ListVersionsByName(_ context.Context, _ string) ([]Version, error) {
	return nil, nil
}
func (f *fakeRegistry) ApplyVersionByName(_ context.Context, _ string, _ int) error { return nil }



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
