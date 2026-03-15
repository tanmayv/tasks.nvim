package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var metaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Output all tags and projects as JSON for LSP autocomplete",
	Run: func(cmd *cobra.Command, args []string) {
		dbConn := getDB()
		defer dbConn.Close()

		type MetaResult struct {
			Tags     []string `json:"tags"`
			Projects []string `json:"projects"`
		}

		result := MetaResult{
			Tags:     []string{},
			Projects: []string{},
		}

		// Fetch unique tags
		tagRows, err := dbConn.DB.Query("SELECT DISTINCT tag_name FROM task_tags ORDER BY tag_name")
		if err == nil {
			for tagRows.Next() {
				var tag string
				if tagRows.Scan(&tag) == nil && tag != "" {
					result.Tags = append(result.Tags, tag)
				}
			}
			tagRows.Close()
		}

		// Fetch unique projects
		projRows, err := dbConn.DB.Query("SELECT DISTINCT project FROM tasks WHERE project != '' ORDER BY project")
		if err == nil {
			for projRows.Next() {
				var proj string
				if projRows.Scan(&proj) == nil && proj != "" {
					result.Projects = append(result.Projects, proj)
				}
			}
			projRows.Close()
		}

		output, err := json.Marshal(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(string(output))
	},
}

func init() {
	rootCmd.AddCommand(metaCmd)
}
