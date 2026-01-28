package primitives

import (
	"testing"
)

func TestMachineConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *MachineConfig
		wantErr bool
	}{
		{
			name: "minimal valid",
			config: &MachineConfig{
				ID:      "machine",
				Initial: "state1",
				States: map[string]*StateConfig{
					"state1": NewStateConfig("state1", Atomic),
				},
			},
			wantErr: false,
		},
		{
			name: "missing machine ID",
			config: &MachineConfig{
				Initial: "state1",
				States: map[string]*StateConfig{
					"state1": NewStateConfig("state1", Atomic),
				},
			},
			wantErr: true,
		},
		{
			name: "missing initial",
			config: &MachineConfig{
				ID: "machine",
				States: map[string]*StateConfig{
					"state1": NewStateConfig("state1", Atomic),
				},
			},
			wantErr: true,
		},
		{
			name: "initial not found",
			config: &MachineConfig{
				ID:      "machine",
				Initial: "missing",
				States: map[string]*StateConfig{
					"state1": NewStateConfig("state1", Atomic),
				},
			},
			wantErr: true,
		},
		{
			name: "empty states",
			config: &MachineConfig{
				ID:      "machine",
				Initial: "state1",
				States:  map[string]*StateConfig{},
			},
			wantErr: true,
		},
		{
			name: "state validation fails",
			config: &MachineConfig{
				ID:      "machine",
				Initial: "bad",
				States: map[string]*StateConfig{
					"bad": NewStateConfig("bad", Atomic).WithInitial("foo"),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid transition target",
			config: &MachineConfig{
				ID:      "machine",
				Initial: "s1",
				States: map[string]*StateConfig{
					"s1": NewStateConfig("s1", Atomic).AddTransition("e", TransitionConfig{Event: "e", Target: "missing"}),
				},
			},
			wantErr: true,
		},
		{
			name: "orphaned state",
			config: &MachineConfig{
				ID:      "machine",
				Initial: "s1",
				States: map[string]*StateConfig{
					"s1":     NewStateConfig("s1", Atomic),
					"orphan": NewStateConfig("orphan", Atomic),
				},
			},
			wantErr: true,
		},
		{
			name: "valid compound hierarchy",
			config: &MachineConfig{
				ID:      "machine",
				Initial: "parent",
				States: map[string]*StateConfig{
					"parent": NewStateConfig("parent", Compound).
						WithInitial("child").
						WithChildren([]*StateConfig{NewStateConfig("child", Atomic)}),
					"child": NewStateConfig("child", Atomic),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
