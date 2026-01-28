package primitives

import (
	"strings"
	"testing"
)

func TestStateConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		newConfig   func() *StateConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid atomic",
			newConfig: func() *StateConfig {
				return NewStateConfig("atomic", Atomic)
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			newConfig: func() *StateConfig {
				return NewStateConfig("", Atomic)
			},
			wantErr:     true,
			errContains: "ID is required",
		},
		{
			name: "invalid type",
			newConfig: func() *StateConfig {
				return NewStateConfig("bad", StateType("invalid"))
			},
			wantErr:     true,
			errContains: "invalid state type",
		},
		{
			name: "atomic with initial",
			newConfig: func() *StateConfig {
				return NewStateConfig("atomic", Atomic).WithInitial("foo")
			},
			wantErr:     true,
			errContains: "cannot have Initial",
		},
		{
			name: "atomic with children",
			newConfig: func() *StateConfig {
				child := NewStateConfig("child", Atomic)
				return NewStateConfig("atomic", Atomic).WithChildren([]*StateConfig{child})
			},
			wantErr:     true,
			errContains: "cannot have Children",
		},
		{
			name: "compound no initial",
			newConfig: func() *StateConfig {
				child := NewStateConfig("child", Atomic)
				return NewStateConfig("compound", Compound).WithChildren([]*StateConfig{child})
			},
			wantErr:     true,
			errContains: "requires Initial child",
		},
		{
			name: "compound invalid initial",
			newConfig: func() *StateConfig {
				return NewStateConfig("compound", Compound).WithInitial("missing").WithChildren([]*StateConfig{NewStateConfig("other", Atomic)})
			},
			wantErr:     true,
			errContains: "initial child \"missing\"",
		},
		{
			name: "valid compound",
			newConfig: func() *StateConfig {
				child := NewStateConfig("child", Atomic)
				return NewStateConfig("compound", Compound).WithInitial("child").WithChildren([]*StateConfig{child})
			},
			wantErr: false,
		},
		{
			name: "valid parallel",
			newConfig: func() *StateConfig {
				child1 := NewStateConfig("ch1", Atomic)
				child2 := NewStateConfig("ch2", Atomic)
				return NewStateConfig("parallel", Parallel).WithInitial("ch1").WithChildren([]*StateConfig{child1, child2})
			},
			wantErr: false,
		},
		{
			name: "history with children",
			newConfig: func() *StateConfig {
				child := NewStateConfig("child", Atomic)
				return NewStateConfig("history", ShallowHistory).WithChildren([]*StateConfig{child})
			},
			wantErr:     true,
			errContains: "cannot have Children",
		},
		{
			name: "valid shallow history",
			newConfig: func() *StateConfig {
				return NewStateConfig("shallow", ShallowHistory)
			},
			wantErr: false,
		},
		{
			name: "valid deep history",
			newConfig: func() *StateConfig {
				return NewStateConfig("deep", DeepHistory)
			},
			wantErr: false,
		},
		{
			name: "empty event name",
			newConfig: func() *StateConfig {
				s := NewStateConfig("s", Atomic)
				s.On = map[string][]TransitionConfig{
					"": {{Event: "e", Target: "t"}},
				}
				return s
			},
			wantErr:     true,
			errContains: "empty event name",
		},
		{
			name: "invalid child recursive",
			newConfig: func() *StateConfig {
				goodChild := NewStateConfig("good", Atomic)
				badChild := NewStateConfig("", Atomic)
				parent := NewStateConfig("parent", Compound).WithInitial("good").WithChildren([]*StateConfig{goodChild, badChild})
				return parent
			},
			wantErr:     true,
			errContains: "ID is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := tt.newConfig()
			err := sc.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf(`Validate() error = "%v", want contains "%s"`, err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}
