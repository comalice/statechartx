// Package production provides production integrations: persistence, event publishing, visualization.
// Implements core interfaces using stdlib where possible.

package production

import (
	"context"
	"encoding/json"
	"gopkg.in/yaml.v3"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/comalice/statechartx/internal/core"
)

// JSONPersister is a stdlib-only file-based persister using JSON serialization.
type JSONPersister struct {
	dir string
}

// NewJSONPersister creates a JSONPersister, ensuring the directory exists.
func NewJSONPersister(dir string) (*JSONPersister, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return &JSONPersister{dir: dir}, nil
}

func (p *JSONPersister) Save(ctx context.Context, snapshot core.MachineSnapshot) error {

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	fn := filepath.Join(p.dir, snapshot.MachineID+".json")
	if err := os.WriteFile(fn, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", fn, err)
	}

	return nil
}

func (p *JSONPersister) Load(ctx context.Context, machineID string) (core.MachineSnapshot, error) {
	fn := filepath.Join(p.dir, machineID+".json")
	data, err := os.ReadFile(fn)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			var empty core.MachineSnapshot
			return empty, fmt.Errorf("machine %q: %w", machineID, os.ErrNotExist)
		}
		return core.MachineSnapshot{}, fmt.Errorf("read %s: %w", fn, err)
	}

	var snapshot core.MachineSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return core.MachineSnapshot{}, fmt.Errorf("json unmarshal: %w", err)
	}
	snapshot.MachineID = machineID // Ensure ID

	return snapshot, nil
}

// YAMLPersister is a file-based persister using YAML serialization for MachineSnapshot.
type YAMLPersister struct {
	dir string
}

// NewYAMLPersister creates a YAMLPersister, ensuring the directory exists.
func NewYAMLPersister(dir string) (*YAMLPersister, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return &YAMLPersister{dir: dir}, nil
}

func (p *YAMLPersister) Save(ctx context.Context, snapshot core.MachineSnapshot) error {
	data, err := yaml.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("yaml marshal: %w", err)
	}

	fn := filepath.Join(p.dir, snapshot.MachineID+".yaml")
	if err := os.WriteFile(fn, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", fn, err)
	}

	return nil
}

func (p *YAMLPersister) Load(ctx context.Context, machineID string) (core.MachineSnapshot, error) {
	fn := filepath.Join(p.dir, machineID+".yaml")
	data, err := os.ReadFile(fn)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			var empty core.MachineSnapshot
			return empty, fmt.Errorf("machine %q: %w", machineID, os.ErrNotExist)
		}
		return core.MachineSnapshot{}, fmt.Errorf("read %s: %w", fn, err)
	}

	var snapshot core.MachineSnapshot
	if err := yaml.Unmarshal(data, &snapshot); err != nil {
		return core.MachineSnapshot{}, fmt.Errorf("yaml unmarshal: %w", err)
	}
	snapshot.MachineID = machineID // Ensure ID
	if err := snapshot.Config.Validate(); err != nil {
		return core.MachineSnapshot{}, fmt.Errorf("config validation after load: %w", err)
	}

	return snapshot, nil
}
