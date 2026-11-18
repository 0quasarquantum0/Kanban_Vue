package kanban

import (
	"encoding/json"
	"testing"
)

func TestCardPatchTellsAbsentFromNull(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantSet     bool
		wantDueDate string // empty means nil
	}{
		{"an absent due date is not set", `{"title":"x"}`, false, ""},
		{"a null due date is set to nil", `{"dueDate":null}`, true, ""},
		{"a string due date is set", `{"dueDate":"2026-12-01"}`, true, "2026-12-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var patch CardPatch
			if err := json.Unmarshal([]byte(tt.json), &patch); err != nil {
				t.Fatal(err)
			}

			if patch.DueDate.Set != tt.wantSet {
				t.Errorf("Set = %v, want %v", patch.DueDate.Set, tt.wantSet)
			}
			switch got := patch.DueDate.Value; {
			case tt.wantDueDate == "" && got != nil:
				t.Errorf("Value = %q, want nil", *got)
			case tt.wantDueDate != "" && (got == nil || *got != tt.wantDueDate):
				t.Errorf("Value = %v, want %q", got, tt.wantDueDate)
			}
		})
	}
}

func TestCardPatchLeavesAbsentFieldsNil(t *testing.T) {
	var patch CardPatch
	if err := json.Unmarshal([]byte(`{"labels":[]}`), &patch); err != nil {
		t.Fatal(err)
	}

	if patch.Title != nil || patch.Description != nil {
		t.Errorf("patch = %+v, want title and description left nil", patch)
	}
	if patch.Labels == nil || len(*patch.Labels) != 0 {
		t.Errorf("Labels = %v, want a pointer to an empty slice", patch.Labels)
	}
}
