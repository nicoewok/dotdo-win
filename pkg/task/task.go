package task

import (
	"slices"
	"time"
)

type Task struct {
	ID     int       `json:"id"`
	Title  string    `json:"title"`
	Status string    `json:"status"` // "todo", "doing", "done"
	Due    time.Time `json:"due,omitempty"`
}

type List struct {
	Tasks []Task `json:"tasks"`
}

// SortByDueDate sorts tasks chronologically by due date.
// Tasks with due dates come first, sorted chronologically.
// Tasks without due dates maintain their relative order after due tasks.
func (l *List) SortByDueDate() {
	slices.SortFunc(l.Tasks, func(a, b Task) int {
		aHasDue := !a.Due.IsZero()
		bHasDue := !b.Due.IsZero()

		// Case 1: Both have due dates - sort chronologically
		if aHasDue && bHasDue {
			if a.Due.Before(b.Due) {
				return -1
			}
			if a.Due.After(b.Due) {
				return 1
			}
			return 0
		}

		// Case 2: Only 'a' has a due date - 'a' comes first
		if aHasDue && !bHasDue {
			return -1
		}

		// Case 3: Only 'b' has a due date - 'b' comes first
		if !aHasDue && bHasDue {
			return 1
		}

		// Case 4: Neither has a due date - keep relative order
		return 0
	})
}

// DeduplicateIDs ensures that all tasks in the list have unique positive IDs.
// If duplicate IDs exist, subsequent occurrences are reassigned starting at maxID + 1.
func (l *List) DeduplicateIDs() bool {
	if len(l.Tasks) == 0 {
		return false
	}

	maxID := 0
	for _, t := range l.Tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}

	seen := make(map[int]bool)
	modified := false

	for i := range l.Tasks {
		id := l.Tasks[i].ID
		if id <= 0 || seen[id] {
			maxID++
			l.Tasks[i].ID = maxID
			seen[maxID] = true
			modified = true
		} else {
			seen[id] = true
		}
	}

	return modified
}

// NextID returns maxID + 1 for assigning a new task ID.
func (l *List) NextID() int {
	maxID := 0
	for _, t := range l.Tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	return maxID + 1
}
