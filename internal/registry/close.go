package registry

// Close closes the underlying DB connection used by the Repository.
func (r *Repository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}
