package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var (
	verbose    int
	bsonFile   string
	metaFile   string
	outputFile string
	version    = "dev"
)

// SetVersion sets the version for the application
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:     "bsontosqlite",
	Version: version,
	Short:   "Convert MongoDB BSON dumps to SQLite database",
	Long:    "A tool to convert MongoDB BSON dump files with metadata.json to SQLite database using modernc sqlite driver.",
	Run:     runConvert,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("bsontosqlite version %s\n", version)
	},
}

func Execute() {
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		slog.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "verbose output (-v for info, -vv for debug)")
	rootCmd.Flags().StringVarP(&bsonFile, "bson", "b", "", "Path to BSON file (required)")
	rootCmd.Flags().StringVarP(&metaFile, "metadata", "m", "", "Path to metadata.json file (required)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "output.db", "Output SQLite database file")

	rootCmd.MarkFlagRequired("bson")
	rootCmd.MarkFlagRequired("metadata")
}

func setupLogging() {
	var level slog.Level
	switch verbose {
	case 0:
		level = slog.LevelWarn
	case 1:
		level = slog.LevelInfo
	default:
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)
}
