// pkg/data/schema_manager.go
package data

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

type SchemaManager struct {
	conn *pgx.Conn
}

func NewSchemaManager(conn *pgx.Conn) *SchemaManager {
	return &SchemaManager{
		conn: conn,
	}
}

func (sm *SchemaManager) InitializeSchema(ctx context.Context) error {
	statements, err := loadSchemaStatements("./sql/schema")
	if err != nil {
		return err
	}

	tx, err := sm.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("executing schema statement: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing schema transaction: %w", err)
	}

	return nil
}

func loadSchemaStatements(schemaDir string) ([]string, error) {
	files, err := os.ReadDir(schemaDir)
	if err != nil {
		return nil, fmt.Errorf("reading schema directory: %w", err)
	}

	fileNames := make([]string, 0, len(files))
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".sql") {
			fileNames = append(fileNames, f.Name())
		}
	}
	sort.Strings(fileNames)

	statements := make([]string, 0, len(fileNames)+1)
	statements = append(statements, `CREATE EXTENSION IF NOT EXISTS pgcrypto`)

	for _, fileName := range fileNames {
		sqlFile := filepath.Join(schemaDir, fileName)
		content, err := os.ReadFile(sqlFile)
		if err != nil {
			return nil, fmt.Errorf("reading schema file %s: %w", fileName, err)
		}
		statements = append(statements, string(content))
	}

	return statements, nil
}
