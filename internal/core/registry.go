// Package core defines the Registry interface for managing multiple Machine instances.
package core

import (
	"context"
	"errors"
	"time"

)

// Registry manages versioned snapshots of running Machine instances.
type Registry interface {
	// Register saves current snapshot with computed version.
	Register(ctx context.Context, machineID string, snapshot MachineSnapshot) error

	// Latest returns the most recent snapshot for machineID.
	Latest(ctx context.Context, machineID string) (MachineSnapshot, error)

	// Version returns snapshot for specific version.
	Version(ctx context.Context, machineID, version string) (MachineSnapshot, error)

	// ListVersions returns versions for machineID, newest first.
	ListVersions(ctx context.Context, machineID string) ([]string, error)

	// ListMachines returns all machine IDs.
	ListMachines(ctx context.Context) ([]string, error)
}

var (
	ErrNotFound     = errors.New("version or machine not found")
	ErrExists       = errors.New("version already exists")
	ErrInvalidState = errors.New("invalid machine state for versioning")
)

// MachineSnapshotVersion annotates snapshot with version.
type MachineSnapshotVersion struct {
	MachineSnapshot
	Version string `json:\"version\" yaml:\"version\"`
	Timestamp time.Time `json:\"timestamp\" yaml:\"timestamp\"`
}
