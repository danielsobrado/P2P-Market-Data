package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSchemaStatements(t *testing.T) {
	schemaDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "002_second.sql"), []byte("SELECT 2;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "001_first.sql"), []byte("SELECT 1;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "003_empty.sql"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "README.txt"), []byte("ignore"), 0o644))

	statements, err := loadSchemaStatements(schemaDir)
	require.NoError(t, err)

	require.Len(t, statements, 4)
	assert.Equal(t, "CREATE EXTENSION IF NOT EXISTS pgcrypto", statements[0])
	assert.Equal(t, "SELECT 1;", statements[1])
	assert.Equal(t, "SELECT 2;", statements[2])
	assert.Equal(t, "", statements[3])
}

func TestLoadSchemaStatementsMissingDirectory(t *testing.T) {
	_, err := loadSchemaStatements(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading schema directory")
}
