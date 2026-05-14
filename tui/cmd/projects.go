package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects with open and due task counts",
	Run: func(cmd *cobra.Command, args []string) {
		dbConn := getDB()
		defer dbConn.Close()

		asJSON, _ := cmd.Flags().GetBool("json")

		stats, err := dbConn.GetProjectStats()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching project stats: %v\n", err)
			os.Exit(1)
		}

		if asJSON {
			output, err := json.MarshalIndent(stats, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(output))
		} else {
			for _, s := range stats {
				fmt.Printf("%d\t%d\t%s\n", s.Due, s.Open, s.Project)
			}
		}
	},
}

func init() {
	projectsCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(projectsCmd)
}
