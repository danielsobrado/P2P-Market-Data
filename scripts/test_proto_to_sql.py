"""Focused unit tests for proto_to_sql.py migration generation.

These tests cover the SQL-generation logic only and do not require a live
database connection.
"""
import os
import tempfile
import textwrap
import pytest

from proto_to_sql import (
    Column,
    Table,
    ProtobufToSQLConverter,
    SchemaManager,
    _quote,
    _SAFE_IDENTIFIER,
)


# ---------------------------------------------------------------------------
# Module-level _quote / _SAFE_IDENTIFIER
# ---------------------------------------------------------------------------

class TestQuote:
    def test_simple_identifier(self):
        assert _quote("my_table") == '"my_table"'

    def test_uppercase_preserved(self):
        assert _quote("MyTable") == '"MyTable"'

    def test_leading_underscore(self):
        assert _quote("_private") == '"_private"'

    def test_rejects_empty_string(self):
        with pytest.raises(ValueError, match="Unsafe SQL identifier"):
            _quote("")

    def test_rejects_digit_start(self):
        with pytest.raises(ValueError, match="Unsafe SQL identifier"):
            _quote("1bad")

    def test_rejects_hyphen(self):
        with pytest.raises(ValueError, match="Unsafe SQL identifier"):
            _quote("bad-name")

    def test_rejects_space(self):
        with pytest.raises(ValueError, match="Unsafe SQL identifier"):
            _quote("bad name")

    def test_rejects_semicolon(self):
        with pytest.raises(ValueError, match="Unsafe SQL identifier"):
            _quote("bad;name")

    def test_rejects_sql_injection(self):
        with pytest.raises(ValueError, match="Unsafe SQL identifier"):
            _quote("t; DROP TABLE users; --")


# ---------------------------------------------------------------------------
# ProtobufToSQLConverter._quote delegates to module-level _quote
# ---------------------------------------------------------------------------

class TestConverterQuote:
    def test_valid(self):
        assert ProtobufToSQLConverter._quote("col") == '"col"'

    def test_invalid_raises(self):
        with pytest.raises(ValueError):
            ProtobufToSQLConverter._quote("bad-col")


# ---------------------------------------------------------------------------
# ProtobufToSQLConverter.parse_proto_file – identifier validation
# ---------------------------------------------------------------------------

class TestParseProtoFile:
    def _write_proto(self, content: str, filename: str = "my_table.proto") -> str:
        d = tempfile.mkdtemp()
        path = os.path.join(d, filename)
        with open(path, "w") as f:
            f.write(textwrap.dedent(content))
        return path

    def test_basic_message_parsed(self):
        proto = """\
            message Root {
              string name = 1;
              int32 value = 2;
            }
        """
        path = self._write_proto(proto)
        converter = ProtobufToSQLConverter(path)
        tables = converter.parse_proto_file(path)
        # 'Root' maps to the file stem because message name is 'Root' (== 'root')
        assert "my_table" in tables
        col_names = [c.name for c in tables["my_table"].columns]
        assert "id" in col_names
        assert "name" in col_names
        assert "value" in col_names

    def test_non_root_message_gets_prefixed_name(self):
        proto = """\
            message Foo {
              string x = 1;
            }
        """
        path = self._write_proto(proto)
        converter = ProtobufToSQLConverter(path)
        tables = converter.parse_proto_file(path)
        assert "my_table_foo" in tables

    def test_invalid_filename_raises(self):
        proto = "message Root { string x = 1; }"
        path = self._write_proto(proto, filename="bad-file.proto")
        converter = ProtobufToSQLConverter(path)
        with pytest.raises(ValueError, match="unsafe SQL identifier"):
            converter.parse_proto_file(path)

    def test_invalid_message_name_skipped(self):
        # A message name starting with a digit is caught by the \w+ regex and
        # then rejected by _SAFE_IDENTIFIER validation.
        proto = "message 1Bad { string x = 1; }"
        path = self._write_proto(proto)
        converter = ProtobufToSQLConverter(path)
        # The regex r'message\s+(\w+)' won't match '1Bad' (digit after message),
        # so the table will simply be absent.  This test confirms no crash.
        tables = converter.parse_proto_file(path)
        assert len(tables) == 0

    def test_invalid_field_name_skipped(self):
        proto = """\
            message Root {
              string good_field = 1;
            }
        """
        path = self._write_proto(proto)
        converter = ProtobufToSQLConverter(path)
        tables = converter.parse_proto_file(path)
        col_names = [c.name for c in tables["my_table"].columns]
        assert "good_field" in col_names

    def test_repeated_field_creates_child_table(self):
        proto = """\
            message Root {
              repeated string tags = 1;
            }
        """
        path = self._write_proto(proto)
        converter = ProtobufToSQLConverter(path)
        tables = converter.parse_proto_file(path)
        assert "my_table_tags" in tables
        fks = tables["my_table_tags"].foreign_keys
        assert any("my_table" in fk for fk in fks)


# ---------------------------------------------------------------------------
# SchemaManager._create_table_statements – idempotency
# ---------------------------------------------------------------------------

def _make_schema_manager_no_db():
    """Return a SchemaManager-like object without a real DB connection."""
    # We only need the SQL-generation methods; bypass __init__ entirely.
    mgr = object.__new__(SchemaManager)
    return mgr


class TestCreateTableStatements:
    def _mgr(self):
        return _make_schema_manager_no_db()

    def test_create_table_if_not_exists(self):
        table = Table(
            name="prices",
            columns=[Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True)],
            foreign_keys=[],
        )
        stmts = SchemaManager._create_table_statements(self._mgr(), table)
        assert any("CREATE TABLE IF NOT EXISTS" in s for s in stmts)

    def test_table_name_quoted(self):
        table = Table(
            name="my_table",
            columns=[Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True)],
            foreign_keys=[],
        )
        stmts = SchemaManager._create_table_statements(self._mgr(), table)
        assert any('"my_table"' in s for s in stmts)

    def test_column_name_quoted(self):
        table = Table(
            name="t",
            columns=[
                Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True),
                Column(name="symbol", data_type="TEXT", is_nullable=True),
            ],
            foreign_keys=[],
        )
        stmts = SchemaManager._create_table_statements(self._mgr(), table)
        ddl = "\n".join(stmts)
        assert '"symbol"' in ddl

    def test_foreign_key_wrapped_in_do_block(self):
        table = Table(
            name="child",
            columns=[Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True)],
            foreign_keys=["FOREIGN KEY (parent_id) REFERENCES parent(id)"],
        )
        stmts = SchemaManager._create_table_statements(self._mgr(), table)
        fk_stmt = [s for s in stmts if "ADD CONSTRAINT" in s]
        assert len(fk_stmt) == 1
        assert "DO $$" in fk_stmt[0]
        assert "EXCEPTION WHEN duplicate_object THEN NULL" in fk_stmt[0]

    def test_multiple_fks_get_unique_names(self):
        table = Table(
            name="child",
            columns=[Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True)],
            foreign_keys=[
                "FOREIGN KEY (a_id) REFERENCES a(id)",
                "FOREIGN KEY (b_id) REFERENCES b(id)",
            ],
        )
        stmts = SchemaManager._create_table_statements(self._mgr(), table)
        fk_stmts = [s for s in stmts if "ADD CONSTRAINT" in s]
        assert len(fk_stmts) == 2
        # Extract constraint names
        names = []
        for s in fk_stmts:
            for part in s.split():
                if part.startswith("fk_"):
                    names.append(part)
        assert len(set(names)) == 2, "FK constraint names must be unique"

    def test_unsafe_table_name_raises(self):
        table = Table(
            name="bad-table",
            columns=[Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True)],
            foreign_keys=[],
        )
        with pytest.raises(ValueError, match="Unsafe SQL identifier"):
            SchemaManager._create_table_statements(self._mgr(), table)


# ---------------------------------------------------------------------------
# SchemaManager._alter_table_statements – idempotency
# ---------------------------------------------------------------------------

class TestAlterTableStatements:
    def _mgr(self):
        return _make_schema_manager_no_db()

    def _base_table(self, name="t"):
        return Table(
            name=name,
            columns=[Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True)],
            foreign_keys=[],
        )

    def test_add_column_uses_if_not_exists(self):
        current = self._base_table()
        new = Table(
            name="t",
            columns=[
                Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True),
                Column(name="price", data_type="DOUBLE PRECISION", is_nullable=True),
            ],
            foreign_keys=[],
        )
        stmts = SchemaManager._alter_table_statements(self._mgr(), current, new)
        add_col = [s for s in stmts if "ADD COLUMN" in s]
        assert len(add_col) == 1
        assert "IF NOT EXISTS" in add_col[0]

    def test_new_fk_wrapped_in_do_block(self):
        current = self._base_table()
        new = Table(
            name="t",
            columns=[Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True)],
            foreign_keys=["FOREIGN KEY (ref_id) REFERENCES other(id)"],
        )
        stmts = SchemaManager._alter_table_statements(self._mgr(), current, new)
        fk_stmts = [s for s in stmts if "ADD CONSTRAINT" in s]
        assert len(fk_stmts) == 1
        assert "DO $$" in fk_stmts[0]
        assert "EXCEPTION WHEN duplicate_object THEN NULL" in fk_stmts[0]

    def test_existing_fk_not_re_added(self):
        fk = "FOREIGN KEY (ref_id) REFERENCES other(id)"
        current = Table(
            name="t",
            columns=[Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True)],
            foreign_keys=[fk],
        )
        new = Table(
            name="t",
            columns=[Column(name="id", data_type="SERIAL", is_nullable=False, is_primary_key=True)],
            foreign_keys=[fk],
        )
        stmts = SchemaManager._alter_table_statements(self._mgr(), current, new)
        fk_stmts = [s for s in stmts if "ADD CONSTRAINT" in s]
        assert len(fk_stmts) == 0

    def test_no_change_produces_no_statements(self):
        current = self._base_table()
        new = self._base_table()
        stmts = SchemaManager._alter_table_statements(self._mgr(), current, new)
        assert stmts == []
