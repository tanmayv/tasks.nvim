package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tanmayv/nvim-task-manager/tui/db"
)

type StatsResponse struct {
	Open   int `json:"open"`
	Closed int `json:"closed"`
	Due    int `json:"due"`
}

var statsCmd = &cobra.Command{
	Use:   "stats [query]",
	Short: "Get stats for tasks matching an optional query (e.g. @project #tag)",
	Run: func(cmd *cobra.Command, args []string) {
		dbConn := getDB()
		defer dbConn.Close()

		asJSON, _ := cmd.Flags().GetBool("json")

		var project string
		var tags []string

		query := strings.Join(args, " ")
		words := strings.Fields(query)
		for _, word := range words {
			if strings.HasPrefix(word, "@") {
				project = strings.TrimPrefix(word, "@")
			} else if strings.HasPrefix(word, "#") {
				tags = append(tags, strings.TrimPrefix(word, "#"))
			}
		}

		opts := db.GetTasksOpts{}
		if project != "" {
			opts.Project = project
		}

		tasks, err := dbConn.GetTasks(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching tasks: %v\n", err)
			os.Exit(1)
		}

		openCount := 0
		closedCount := 0
		dueCount := 0

		todayStr := time.Now().UTC().Format("2006-01-02")

		for _, t := range tasks {
			if len(tags) > 0 {
				hasAllTags := true
				for _, tag := range tags {
					found := false
					for _, tTag := range t.Tags {
						if tag == tTag {
							found = true
							break
						}
					}
					if !found {
						hasAllTags = false
						break
					}
				}
				if !hasAllTags {
					continue
				}
			}

			if t.Status == "todo" || t.Status == "in_progress" {
				openCount++
				if t.DueDate != "" && t.DueDate <= todayStr {
					dueCount++
				}
			} else if t.Status == "done" || t.Status == "cancelled" {
				closedCount++
			}
		}

		if asJSON {
			resp := StatsResponse{
				Open:   openCount,
				Closed: closedCount,
				Due:    dueCount,
			}
			output, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(output))
		} else {
			fmt.Printf("Open: %d, Closed: %d, Due: %d\n", openCount, closedCount, dueCount)
		}
	},
}

func init() {
	statsCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(statsCmd)
}
