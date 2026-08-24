package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const CredentialsFileEnvironment = "ESPER_CREDS_FILE"

type Config struct {
	Environment  string `json:"environment"`
	APIKey       string `json:"api_key"`
	EnterpriseID string `json:"enterprise_id"`
}

type ActiveResource struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type ActiveContext struct {
	Device      *ActiveResource `json:"device,omitempty"`
	Application *ActiveResource `json:"application,omitempty"`
	Group       *ActiveResource `json:"group,omitempty"`
}

type State struct {
	Config Config        `json:"config"`
	Active ActiveContext `json:"active"`
}

type StateStore struct {
	Path string
}

func DefaultCredentialsPath() (string, error) {
	if path := os.Getenv(CredentialsFileEnvironment); path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".esper", "db", "creds.json"), nil
}

func NewStateStore() (*StateStore, error) {
	path, err := DefaultCredentialsPath()
	if err != nil {
		return nil, err
	}
	return &StateStore{Path: path}, nil
}

func (store *StateStore) Load() (State, error) {
	data, err := os.ReadFile(store.Path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read state: %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("decode state: %w", err)
	}
	return state, nil
}

func (store *StateStore) Save(state State) error {
	directory := filepath.Dir(store.Path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("set state directory permissions: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	data = append(data, '\n')

	temporary, err := os.CreateTemp(directory, ".creds-*")
	if err != nil {
		return fmt.Errorf("create temporary state: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("set temporary state permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary state: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary state: %w", err)
	}
	if err := os.Rename(temporaryPath, store.Path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	if err := os.Chmod(store.Path, 0o600); err != nil {
		return fmt.Errorf("set state permissions: %w", err)
	}
	return nil
}
