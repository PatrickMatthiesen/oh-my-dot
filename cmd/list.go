package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	listCommand.Flags().BoolP("verbose", "v", false, "List all files and their linkings (TODO)")

	listCommand.Flags().Bool("json", false, "Print versioned JSON file and destination records")
	rootCmd.AddCommand(listCommand)
}

var listCommand = &cobra.Command{
	Aliases:      []string{"ls"},
	Use:          "list",
	Short:        "List all files in the repository",
	Long:         `List all files in the repository.`,
	GroupID:      "basics",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := git.GetRepository(viper.GetString("repo-path")); err != nil {
			return fmt.Errorf("inspect repository: %w", err)
		}
		files, err := git.ListFiles()
		if err != nil {
			return fmt.Errorf("list files: %w", err)
		}
		sort.Strings(files)
		linkings, err := symlink.GetLinkings()
		if err != nil {
			return fmt.Errorf("read linkings: %w", err)
		}
		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			type fileRecord struct {
				Name        string `json:"name"`
				Destination string `json:"destination,omitempty"`
			}
			records := make([]fileRecord, 0, len(files))
			for _, file := range files {
				records = append(records, fileRecord{Name: file, Destination: linkings[file]})
			}
			report := struct {
				SchemaVersion int          `json:"schemaVersion"`
				Files         []fileRecord `json:"files"`
			}{1, records}
			if err := json.NewEncoder(cmd.OutOrStdout()).Encode(report); err != nil {
				return fmt.Errorf("write file JSON: %w", err)
			}
			return nil
		}
		if len(files) == 0 {
			fileops.ColorPrintln("No files in repository", fileops.Yellow)
			return nil
		}
		fileops.ColorPrintfn(fileops.Cyan, "Files in repository:")
		for _, file := range files {
			if linked, ok := linkings[file]; ok {
				fmt.Println(fileops.SColorPrint(file, fileops.Green) + " -> " + fileops.SColorPrint(linked, fileops.Blue))
			} else {
				fileops.ColorPrintfn(fileops.Yellow, "%s (not linked)", file)
			}
		}
		return nil
	},
}
