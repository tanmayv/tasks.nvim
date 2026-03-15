package parser

import (
	"reflect"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want *Task
	}{
		{
			name: "Simple todo",
			line: "- [ ] Buy milk",
			want: &Task{
				Status:       "todo",
				Description:  "Buy milk",
				OriginalLine: "- [ ] Buy milk",
				Prefix:       "- ",
				Tags:         []string{},
				Metadata:     make(map[string]string),
			},
		},
		{
			name: "Complex task",
			line: "  - [/] Fix bug | @work #urgent +high b:123 id:t:abc123 due:today",
			want: &Task{
				Status:       "in_progress",
				Description:  "Fix bug",
				OriginalLine: "  - [/] Fix bug | @work #urgent +high b:123 id:t:abc123 due:today",
				Prefix:       "  - ",
				Project:      "work",
				Tags:         []string{"urgent"},
				Priority:     "high",
				DueDate:      "today",
				ID:           "t:abc123",
				Metadata:     map[string]string{"b": "123"},
			},
		},
		{
			name: "Not a task",
			line: "Just some text",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLine(tt.line)
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseLine() = %v, want %v", got, tt.want)
				}
				return
			}
			if got == nil {
				t.Fatalf("ParseLine() = nil, want %v", tt.want)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.want.Status)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %v, want %v", got.Description, tt.want.Description)
			}
			if got.Project != tt.want.Project {
				t.Errorf("Project = %v, want %v", got.Project, tt.want.Project)
			}
			if !reflect.DeepEqual(got.Tags, tt.want.Tags) {
				t.Errorf("Tags = %v, want %v", got.Tags, tt.want.Tags)
			}
		})
	}
}

func TestFormatLine(t *testing.T) {
	task := &Task{
		Status:      "done",
		Description: "Fix bug",
		Prefix:      "- ",
		Project:     "work",
		Tags:        []string{"urgent"},
		Priority:    "high",
		DueDate:     "today",
		ID:          "t:abc123",
		Metadata:    map[string]string{"b": "123"},
	}

	got := FormatLine(task)
	// Map iteration is ordered here because we sort keys
	want := "- [x] Fix bug | @work #urgent +high due:today b:123 id:t:abc123"

	if got != want {
		t.Errorf("FormatLine() = %v, want %v", got, want)
	}
}
