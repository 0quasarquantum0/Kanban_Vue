package kanban

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestStateJSONUsesTheFrontendFieldNames(t *testing.T) {
	due := "2026-11-20"
	state := State{
		Boards:  map[string]Board{"b": {ID: "b", Title: "Board", ColumnIDs: []string{"k"}}},
		Columns: map[string]Column{"k": {ID: "k", Title: "Column", CardIDs: []string{"c"}}},
		Cards: map[string]Card{"c": {
			ID: "c", Title: "Card", Description: "d", Labels: []string{"bug"},
			DueDate: &due, CreatedAt: "2026-11-18T00:00:00.000Z",
		}},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}

	const want = `{"boards":{"b":{"id":"b","title":"Board","columnIds":["k"]}},` +
		`"columns":{"k":{"id":"k","title":"Column","cardIds":["c"]}},` +
		`"cards":{"c":{"id":"c","title":"Card","description":"d","labels":["bug"],` +
		`"dueDate":"2026-11-20","createdAt":"2026-11-18T00:00:00.000Z"}}}`
	if string(data) != want {
		t.Errorf("JSON =\n%s\nwant\n%s", data, want)
	}
}

func TestCardWithoutDueDateEncodesNull(t *testing.T) {
	data, err := json.Marshal(Card{ID: "c", Labels: []string{}})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), `"dueDate":null`) {
		t.Errorf("JSON = %s, want dueDate to be null", data)
	}
	if !strings.Contains(string(data), `"labels":[]`) {
		t.Errorf("JSON = %s, want labels to be []", data)
	}
}

func TestEmptyStateEncodesObjectsNotNull(t *testing.T) {
	data, err := json.Marshal(State{})
	if err != nil {
		t.Fatal(err)
	}

	if want := `{"boards":{},"columns":{},"cards":{}}`; string(data) != want {
		t.Errorf("JSON = %s, want %s", data, want)
	}
}

func TestStateJSONKeepsTheOrderOfBoards(t *testing.T) {
	state := State{
		Boards: map[string]Board{
			"a": {ID: "a", ColumnIDs: []string{}},
			"b": {ID: "b", ColumnIDs: []string{}},
			"c": {ID: "c", ColumnIDs: []string{}},
		},
		BoardOrder: []string{"c", "a", "b"},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var back State
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}

	if want := []string{"c", "a", "b"}; !slices.Equal(back.BoardOrder, want) {
		t.Errorf("board order after a round trip = %v, want %v", back.BoardOrder, want)
	}
	if i, j := strings.Index(string(data), `"c":`), strings.Index(string(data), `"a":`); i > j {
		t.Errorf("board c should be written before board a: %s", data)
	}
}

func TestBoardIDsFallsBackToSortedOrder(t *testing.T) {
	state := State{
		Boards:     map[string]Board{"b": {}, "a": {}, "c": {}},
		BoardOrder: []string{"c", "gone", "c"},
	}

	got := state.BoardIDs()

	if want := []string{"c", "a", "b"}; !slices.Equal(got, want) {
		t.Errorf("BoardIDs() = %v, want %v", got, want)
	}
}

func TestStateRoundTripsThroughJSON(t *testing.T) {
	want := NewSeedState()

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got State
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("state changed in a JSON round trip:\n got %+v\nwant %+v", got, want)
	}
}
