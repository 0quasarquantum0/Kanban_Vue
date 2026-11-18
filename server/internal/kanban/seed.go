package kanban

import "time"

// timestampLayout is how the frontend writes dates (JavaScript's toISOString()).
const timestampLayout = "2006-01-02T15:04:05.000Z"

// Now returns the current time as an ISO 8601 string in UTC, in the frontend's format.
func Now() string {
	return time.Now().UTC().Format(timestampLayout)
}

// cardSeed is the part of a card that the demo data spells out.
type cardSeed struct {
	title       string
	description string
	labels      []string
	dueDate     string // empty means no due date
}

var columnSeeds = []struct {
	title string
	cards []cardSeed
}{
	{"To do", []cardSeed{
		{
			title:       "Drag and drop between columns",
			description: "Native HTML5 drag events wrapped in a composable.",
			labels:      []string{"feature"},
			dueDate:     "2026-11-20",
		},
		{title: "Persist the board to localStorage", labels: []string{"feature"}},
		{
			title:       "Card title overflows on narrow screens",
			description: "Long words without spaces push the card wider than its column.",
			labels:      []string{"bug", "ui"},
			dueDate:     "2026-11-14",
		},
	}},
	{"Doing", []cardSeed{
		{
			title:       "Build the static board",
			description: "BoardView, KanbanColumn and KanbanCard with props and slots.",
			labels:      []string{"ui"},
			dueDate:     "2026-11-12",
		},
		{title: "Read the Vue docs on reactivity", labels: []string{"learning"}},
	}},
	{"Done", []cardSeed{
		{
			title:       "Scaffold the project",
			description: "create-vue with TypeScript, Router, Pinia and Vitest.",
			labels:      []string{"chore"},
		},
	}},
}

// NewSeedState builds the demo state, with new IDs on every call. It mirrors createSeedState()
// in src/seed.ts.
func NewSeedState() State {
	state := State{
		Boards:  map[string]Board{},
		Columns: map[string]Column{},
		Cards:   map[string]Card{},
	}
	createdAt := Now()

	columnIDs := make([]string, 0, len(columnSeeds))
	for _, columnSeed := range columnSeeds {
		cardIDs := make([]string, 0, len(columnSeed.cards))
		for _, seed := range columnSeed.cards {
			card := Card{
				ID:          NewID(),
				Title:       seed.title,
				Description: seed.description,
				Labels:      seed.labels,
				CreatedAt:   createdAt,
			}
			if seed.dueDate != "" {
				card.DueDate = &seed.dueDate
			}
			state.Cards[card.ID] = card
			cardIDs = append(cardIDs, card.ID)
		}

		column := Column{ID: NewID(), Title: columnSeed.title, CardIDs: cardIDs}
		state.Columns[column.ID] = column
		columnIDs = append(columnIDs, column.ID)
	}

	board := Board{ID: NewID(), Title: "Vue Kanban", ColumnIDs: columnIDs}
	state.Boards[board.ID] = board
	state.BoardOrder = []string{board.ID}
	return state
}
