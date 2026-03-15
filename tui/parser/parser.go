package parser

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tanmayv/nvim-task-manager/tui/date"
)

type Task struct {
	ID           string
	Description  string
	Status       string
	Project      string
	Tags         []string
	Priority     string
	DueDate      string
	StartDate    string
	Metadata     map[string]string
	OriginalLine string
	Prefix       string // The part before "[ ] " like "- " or "  - "
}

var statusMap = map[string]string{
	" ": "todo",
	"x": "done",
	"/": "in_progress",
	"-": "cancelled",
}

var reverseStatusMap = map[string]string{
	"todo":        " ",
	"done":        "x",
	"in_progress": "/",
	"cancelled":   "-",
}

var taskRegex = regexp.MustCompile(`^(\s*[\-*]\s+)\[([ x/\-])\]\s+(.*)$`)

func GenerateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	id := "t:"
	for i := 0; i < 6; i++ {
		id += string(chars[rand.Intn(len(chars))])
	}
	return id
}

func ParseLine(line string) *Task {
	matches := taskRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	prefix := matches[1]
	statusChar := matches[2]
	rest := matches[3]

	task := &Task{
		Status:       statusMap[statusChar],
		Tags:         []string{},
		Metadata:     make(map[string]string),
		OriginalLine: line,
		Prefix:       prefix,
	}

	words := strings.Fields(rest)
	var descriptionParts []string

	for _, word := range words {
		if word == "|" {
			continue
		} else if strings.HasPrefix(word, "id:") {
			task.ID = strings.TrimPrefix(word, "id:")
		} else if strings.HasPrefix(word, "@") {
			task.Project = strings.TrimPrefix(word, "@")
		} else if strings.HasPrefix(word, "#") {
			task.Tags = append(task.Tags, strings.TrimPrefix(word, "#"))
		} else if strings.HasPrefix(word, "tag:") {
			task.Tags = append(task.Tags, strings.TrimPrefix(word, "tag:"))
		} else if strings.HasPrefix(word, "+") {
			task.Priority = strings.TrimPrefix(word, "+")
		} else if strings.Contains(word, ":") {
			parts := strings.SplitN(word, ":", 2)
			key, val := parts[0], parts[1]
			if key == "due" {
				task.DueDate = date.ParseRelative(val)
			} else if key == "start" {
				task.StartDate = date.ParseRelative(val)
			} else {
				task.Metadata[key] = val
			}
		} else {
			descriptionParts = append(descriptionParts, word)
		}
	}

	task.Description = strings.Join(descriptionParts, " ")
	return task
}

func FormatDescription(task *Task) []string {
	parts := []string{task.Description}

	if task.Project != "" {
		parts = append(parts, "@"+task.Project)
	}
	for _, tag := range task.Tags {
		parts = append(parts, "#"+tag)
	}
	if task.Priority != "" {
		parts = append(parts, "+"+task.Priority)
	}
	if task.DueDate != "" {
		parts = append(parts, "due:"+task.DueDate)
	}
	if task.StartDate != "" {
		parts = append(parts, "start:"+task.StartDate)
	}

	// Sort metadata for deterministic output
	var keys []string
	for k := range task.Metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%s", k, task.Metadata[k]))
	}

	if task.ID != "" {
		parts = append(parts, "id:"+task.ID)
	}

	return parts
}

func FormatLine(task *Task) string {
	statusChar, ok := reverseStatusMap[task.Status]
	if !ok {
		statusChar = " "
	}

	parts := FormatDescription(task)
	desc := parts[0]
	rest := parts[1:]

	line := fmt.Sprintf("%s[%s] %s", task.Prefix, statusChar, desc)
	if len(rest) > 0 {
		line += " | " + strings.Join(rest, " ")
	}

	return line
}
