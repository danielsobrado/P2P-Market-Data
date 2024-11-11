// pkg/data/schema.go
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

func initSchema(ctx context.Context, conn *pgx.Conn) error {
	schemaDir := "./sql/schema"
	files, err := os.ReadDir(schemaDir)
	if err != nil {
		return fmt.Errorf("reading schema directory: %w", err)
	}

	// Sort files by name to ensure consistent order
	fileNames := make([]string, 0, len(files))
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".sql") {
			fileNames = append(fileNames, f.Name())
		}
	}
	sort.Strings(fileNames)

	// Execute each schema file in transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, fileName := range fileNames {
		sqlFile := filepath.Join(schemaDir, fileName)
		content, err := os.ReadFile(sqlFile)
		if err != nil {
			return fmt.Errorf("reading schema file %s: %w", fileName, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("executing schema file %s: %w", fileName, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing schema transaction: %w", err)
	}

	return nil
}
