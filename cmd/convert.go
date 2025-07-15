package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	_ "modernc.org/sqlite"
)

type Metadata struct {
	Database       string                 `json:"database"`
	Collection     string                 `json:"collection"`
	CollectionName string                 `json:"collectionName"`
	Type           string                 `json:"type"`
	UUID           string                 `json:"uuid"`
	Metadata       map[string]interface{} `json:"metadata"`
	Indexes        []Index                `json:"indexes"`
}

type Index struct {
	V    interface{}            `json:"v"`
	Key  map[string]interface{} `json:"key"`
	Name string                 `json:"name"`
}

func runConvert(cmd *cobra.Command, args []string) {
	setupLogging()

	slog.Info("Starting BSON to SQLite conversion",
		"bson_file", bsonFile,
		"metadata_file", metaFile,
		"output_file", outputFile)

	metadata, err := parseMetadata(metaFile)
	if err != nil {
		slog.Error("Failed to parse metadata", "error", err)
		os.Exit(1)
	}

	db, err := createDatabase(outputFile, metadata)
	if err != nil {
		slog.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := importBSONData(db, bsonFile, metadata); err != nil {
		slog.Error("Failed to import BSON data", "error", err)
		os.Exit(1)
	}

	slog.Info("Conversion completed successfully")
}

func parseMetadata(filepath string) (*Metadata, error) {
	slog.Debug("Parsing metadata file", "file", filepath)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	collectionName := metadata.Collection
	if collectionName == "" {
		collectionName = metadata.CollectionName
	}

	slog.Info("Metadata parsed successfully",
		"database", metadata.Database,
		"collection", collectionName)

	return &metadata, nil
}

func createDatabase(filepath string, metadata *Metadata) (*sql.DB, error) {
	slog.Debug("Creating SQLite database", "file", filepath)

	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	collectionName := metadata.Collection
	if collectionName == "" {
		collectionName = metadata.CollectionName
	}
	tableName := sanitizeTableName(collectionName)
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT UNIQUE,
			data TEXT
		)
	`, tableName)

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	slog.Info("Database table created", "table", tableName)
	return db, nil
}

func importBSONData(db *sql.DB, filepath string, metadata *Metadata) error {
	slog.Debug("Reading BSON file", "file", filepath)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read BSON file: %w", err)
	}

	collectionName := metadata.Collection
	if collectionName == "" {
		collectionName = metadata.CollectionName
	}
	tableName := sanitizeTableName(collectionName)
	insertSQL := fmt.Sprintf(`INSERT OR REPLACE INTO %s (document_id, data) VALUES (?, ?)`, tableName)

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	offset := 0
	count := 0

	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		docSize := int(data[offset]) | int(data[offset+1])<<8 | int(data[offset+2])<<16 | int(data[offset+3])<<24

		if offset+docSize > len(data) {
			slog.Warn("Incomplete document at end of file", "offset", offset, "expected_size", docSize)
			break
		}

		docData := data[offset : offset+docSize]

		var doc bson.M
		if err := bson.Unmarshal(docData, &doc); err != nil {
			slog.Warn("Failed to unmarshal BSON document", "offset", offset, "error", err)
			offset += docSize
			continue
		}

		docID := ""
		if id, ok := doc["_id"]; ok {
			docID = fmt.Sprintf("%v", id)
		}

		jsonData, err := json.Marshal(doc)
		if err != nil {
			slog.Warn("Failed to marshal document to JSON", "doc_id", docID, "error", err)
			offset += docSize
			continue
		}

		if _, err := stmt.Exec(docID, string(jsonData)); err != nil {
			slog.Warn("Failed to insert document", "doc_id", docID, "error", err)
		} else {
			count++
			if count%1000 == 0 {
				slog.Info("Progress", "documents_processed", count)
			}
		}

		offset += docSize
	}

	slog.Info("BSON import completed", "total_documents", count)
	return nil
}

func sanitizeTableName(name string) string {
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}
