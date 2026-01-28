package primitives

import (
	"strings"
	"testing"
)

func TestTransitionConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		tc          TransitionConfig
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid",
			tc:      TransitionConfig{Event: "click", Target: "next"},
			wantErr: false,
		},
		{
			name:        "missing event",
			tc:          TransitionConfig{Target: "next"},
			wantErr:     true,
			errContains: "event is required",
		},
		{
			name:        "missing target",
			tc:          TransitionConfig{Event: "click"},
			wantErr:     true,
			errContains: "target is required",
		},
		{
			name:        "negative priority",
			tc:          TransitionConfig{Event: "e", Target: "t", Priority: -1},
			wantErr:     true,
			errContains: "non-negative",
		},
		{
			name:        "empty target segment",
			tc:          TransitionConfig{Event: "e", Target: "parent..child"},
			wantErr:     true,
			errContains: "empty segment",
		},
		{
			name:        "invalid target char",
			tc:          TransitionConfig{Event: "e", Target: "invalid@state"},
			wantErr:     true,
			errContains: "invalid character",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tc.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf(`error "%v" does not contain "%s"`, err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSortTransitions(t *testing.T) {
	trans := []TransitionConfig{
		{Event: "event", Target: "low_prio", Priority: 1},
		{Event: "event", Target: "high_prio", Priority: 10},
		{Event: "event", Target: "med_prio", Priority: 5},
		{Event: "event", Target: "default", Priority: 0},
	}
	expectedTargets := []string{"high_prio", "med_prio", "low_prio", "default"}
	SortTransitions(trans)
	for i, exp := range expectedTargets {
		if trans[i].Target != exp {
			t.Errorf("SortTransitions[%d]: got Target=%q want %q", i, trans[i].Target, exp)
		}
	}
}
