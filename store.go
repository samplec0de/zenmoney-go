package zenmoney

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Store persists token and synced data to ~/.config/zenmoney/.
type Store struct {
	dir string
}

// NewStore creates a store at the default config location.
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".config", "zenmoney")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// SaveToken persists the OAuth token.
func (s *Store) SaveToken(tok *Token) error {
	return s.writeJSON("token.json", tok)
}

// LoadToken loads the stored OAuth token.
func (s *Store) LoadToken() (*Token, error) {
	var tok Token
	if err := s.readJSON("token.json", &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

// SyncState holds the last sync result and server timestamp.
type SyncState struct {
	ServerTimestamp int64            `json:"serverTimestamp"`
	Instrument     []Instrument     `json:"instrument,omitempty"`
	Company        []Company        `json:"company,omitempty"`
	User           []User           `json:"user,omitempty"`
	Account        []Account        `json:"account,omitempty"`
	Tag            []Tag            `json:"tag,omitempty"`
	Merchant       []Merchant       `json:"merchant,omitempty"`
	Reminder       []Reminder       `json:"reminder,omitempty"`
	ReminderMarker []ReminderMarker `json:"reminderMarker,omitempty"`
	Transaction    []Transaction    `json:"transaction,omitempty"`
	Budget         []Budget         `json:"budget,omitempty"`
}

// SaveSync persists the sync state.
func (s *Store) SaveSync(state *SyncState) error {
	return s.writeJSON("data.json", state)
}

// LoadSync loads the stored sync state.
func (s *Store) LoadSync() (*SyncState, error) {
	var state SyncState
	if err := s.readJSON("data.json", &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// MergeSync applies an incremental diff response into the existing state.
// New/updated entities replace old ones by ID; deletions are removed.
func (s *SyncState) MergeSync(diff *DiffResponse) {
	s.ServerTimestamp = diff.ServerTimestamp
	s.Instrument = mergeByID(s.Instrument, diff.Instrument, func(v Instrument) int { return v.ID })
	s.Company = mergeByID(s.Company, diff.Company, func(v Company) int { return v.ID })
	s.User = mergeByID(s.User, diff.User, func(v User) int { return v.ID })
	s.Account = mergeByKey(s.Account, diff.Account, func(v Account) string { return v.ID })
	s.Tag = mergeByKey(s.Tag, diff.Tag, func(v Tag) string { return v.ID })
	s.Merchant = mergeByKey(s.Merchant, diff.Merchant, func(v Merchant) string { return v.ID })
	s.Reminder = mergeByKey(s.Reminder, diff.Reminder, func(v Reminder) string { return v.ID })
	s.ReminderMarker = mergeByKey(s.ReminderMarker, diff.ReminderMarker, func(v ReminderMarker) string { return v.ID })
	s.Transaction = mergeByKey(s.Transaction, diff.Transaction, func(v Transaction) string { return v.ID })

	// Budgets keyed by user+tag+date
	s.Budget = mergeBudgets(s.Budget, diff.Budget)

	// Apply hard deletions
	for _, d := range diff.Deletion {
		s.applyDeletion(d)
	}
}

func (s *SyncState) applyDeletion(d Deletion) {
	switch d.Object {
	case "account":
		s.Account = deleteByKey(s.Account, d.ID, func(v Account) string { return v.ID })
	case "tag":
		s.Tag = deleteByKey(s.Tag, d.ID, func(v Tag) string { return v.ID })
	case "merchant":
		s.Merchant = deleteByKey(s.Merchant, d.ID, func(v Merchant) string { return v.ID })
	case "transaction":
		s.Transaction = deleteByKey(s.Transaction, d.ID, func(v Transaction) string { return v.ID })
	case "reminder":
		s.Reminder = deleteByKey(s.Reminder, d.ID, func(v Reminder) string { return v.ID })
	case "reminderMarker":
		s.ReminderMarker = deleteByKey(s.ReminderMarker, d.ID, func(v ReminderMarker) string { return v.ID })
	}
}

func mergeByID[T any](existing, incoming []T, id func(T) int) []T {
	if len(incoming) == 0 {
		return existing
	}
	m := make(map[int]T, len(existing))
	for _, v := range existing {
		m[id(v)] = v
	}
	for _, v := range incoming {
		m[id(v)] = v
	}
	result := make([]T, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

func mergeByKey[T any](existing, incoming []T, key func(T) string) []T {
	if len(incoming) == 0 {
		return existing
	}
	m := make(map[string]T, len(existing))
	for _, v := range existing {
		m[key(v)] = v
	}
	for _, v := range incoming {
		m[key(v)] = v
	}
	result := make([]T, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

func deleteByKey[T any](slice []T, id string, key func(T) string) []T {
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if key(v) != id {
			result = append(result, v)
		}
	}
	return result
}

func mergeBudgets(existing, incoming []Budget) []Budget {
	if len(incoming) == 0 {
		return existing
	}
	type bkey struct {
		user int
		tag  string
		date string
	}
	tagStr := func(b Budget) string {
		if b.Tag == nil {
			return ""
		}
		return *b.Tag
	}
	m := make(map[bkey]Budget, len(existing))
	for _, b := range existing {
		m[bkey{b.User, tagStr(b), b.Date}] = b
	}
	for _, b := range incoming {
		m[bkey{b.User, tagStr(b), b.Date}] = b
	}
	result := make([]Budget, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

func (s *Store) writeJSON(name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", name, err)
	}
	return os.WriteFile(filepath.Join(s.dir, name), data, 0600)
}

func (s *Store) readJSON(name string, v any) error {
	data, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
