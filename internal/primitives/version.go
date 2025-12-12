// Package primitives provides versioning utilities for MachineConfig.
package primitives

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// ComputeVersion computes deterministic version for MachineConfig.
// Priority: user-provided config.Version, else SHA256(config JSON)[:8] + timestamp.
func ComputeVersion(config *MachineConfig) string {
	if config.Version != "" {
		return config.Version
	}

	data, err := json.Marshal(config)
	if err != nil {
		// Fallback (should not happen for valid config)
		return fmt.Sprintf("invalid-%d", time.Now().Unix())
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x-%s", hash[:8], time.Now().UTC().Format("20060102T150405Z"))
}

// VersionFromSnapshot removed to avoid import cycle; use primitives.ComputeVersion(&snapshot.Config) in core
