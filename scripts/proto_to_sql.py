from typing import Dict, List, Optional, Set, Tuple
import os
import re
import glob
import psycopg2
import yaml
import datetime
from psycopg2.extensions import connection
from dataclasses import dataclass
import logging
from pathlib import Path

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

@dataclass
class Column:
    name: str
    data_type: str
    is_nullable: bool
    is_primary_key: bool = False

@dataclass
class Table:
    name: str
    columns: List[Column]
    foreign_keys: List[str]

class ConfigManager:
    @staticmethod
    def load_db_config() -> Dict[str, str]:
        """Load database configuration from YAML file."""
        config_path = Path(__file__).parent.parent / 'config' / 'db_config.yaml'
        try:
            with open(config_path, 'r') as f:
                config = yaml.safe_load(f)
                return {
                    'dbname': config['database']['name'],
                    'user': config['database']['user'],
                    'password': config['database']['password'],
                    'host': config['database']['host'],
                    'port': config['database']['port']
                }
        except Exception as e:
            logger.error(f"Failed to load database configuration: {e}")
            raise

class SQLFileManager:
    def __init__(self, sql_dir: Path):
        self.sql_dir = sql_dir
        self.sql_dir.mkdir(parents=True, exist_ok=True)

    def write_migration(self, proto_name: str, statements: List[str]) -> Path:
        """Write SQL migration statements to a file."""
        if not statements:
            return None

        timestamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"{timestamp}_{proto_name}_migration.sql"
        filepath = self.sql_dir / filename

        with open(filepath, 'w') as f:
            f.write("-- Auto-generated migration from protobuf\n")
            f.write(f"-- Proto: {proto_name}\n")
            f.write(f"-- Generated at: {datetime.datetime.now().isoformat()}\n\n")
            f.write("BEGIN;\n\n")
            for statement in statements:
                f.write(f"{statement}\n")
            f.write("\nCOMMIT;\n")

        return filepath

class DatabaseSchema:
    def __init__(self, conn: connection):
        self.conn = conn
        self.current_tables = self._get_current_schema()

    def _get_current_schema(self) -> Dict[str, Table]:
        """Fetch current database schema."""
        tables = {}
        with self.conn.cursor() as cur:
            # Get all tables in the public schema
            cur.execute("""
                SELECT table_name 
                FROM information_schema.tables 
                WHERE table_schema = 'public'
            """)
            db_tables = cur.fetchall()

            for (table_name,) in db_tables:
                # Get column information
                cur.execute("""
                    SELECT 
                        column_name,
                        data_type,
                        is_nullable,
                        CASE 
                            WHEN pk.column_name IS NOT NULL THEN true 
                            ELSE false 
                        END as is_primary_key
                    FROM information_schema.columns c
                    LEFT JOIN (
                        SELECT ku.column_name
                        FROM information_schema.table_constraints tc
                        JOIN information_schema.key_column_usage ku
                            ON tc.constraint_name = ku.constraint_name
                        WHERE tc.constraint_type = 'PRIMARY KEY'
                        AND tc.table_name = %s
                    ) pk ON c.column_name = pk.column_name
                    WHERE table_name = %s
                """, (table_name, table_name))
                
                columns = [
                    Column(
                        name=row[0],
                        data_type=row[1].upper(),
                        is_nullable=row[2] == 'YES',
                        is_primary_key=row[3]
                    )
                    for row in cur.fetchall()
                ]

                # Get foreign key constraints
                cur.execute("""
                    SELECT
                        tc.constraint_name,
                        kcu.column_name,
                        ccu.table_name AS foreign_table_name,
                        ccu.column_name AS foreign_column_name
                    FROM information_schema.table_constraints AS tc
                    JOIN information_schema.key_column_usage AS kcu
                        ON tc.constraint_name = kcu.constraint_name
                    JOIN information_schema.constraint_column_usage AS ccu
                        ON ccu.constraint_name = tc.constraint_name
                    WHERE tc.constraint_type = 'FOREIGN KEY'
                    AND tc.table_name = %s
                """, (table_name,))
                
                foreign_keys = [
                    f"FOREIGN KEY ({row[1]}) REFERENCES {row[2]}({row[3]})"
                    for row in cur.fetchall()
                ]

                tables[table_name] = Table(
                    name=table_name,
                    columns=columns,
                    foreign_keys=foreign_keys
                )

        return tables

class ProtobufToSQLConverter:
    TYPE_MAPPING = {
        'double': 'DOUBLE PRECISION',
        'float': 'REAL',
        'int32': 'INTEGER',
        'int64': 'BIGINT',
        'uint32': 'INTEGER',
        'uint64': 'BIGINT',
        'sint32': 'INTEGER',
        'sint64': 'BIGINT',
        'fixed32': 'INTEGER',
        'fixed64': 'BIGINT',
        'sfixed32': 'INTEGER',
        'sfixed64': 'BIGINT',
        'bool': 'BOOLEAN',
        'string': 'TEXT',
        'bytes': 'BYTEA',
        'timestamp': 'TIMESTAMP',
    }

    def __init__(self, proto_path: str):
        self.proto_path = proto_path
        self.tables: Dict[str, Table] = {}
        self.current_message = None

    def parse_proto_file(self, proto_file: str) -> Dict[str, Table]:
        """Parse a single proto file and return table definitions."""
        self.tables = {}
        self.current_message = None
        
        # Use the proto filename (without extension) as the table name base
        table_name_base = os.path.splitext(os.path.basename(proto_file))[0].lower()
        
        with open(proto_file, 'r') as f:
            content = f.read()
            
        # Parse message definitions
        message_pattern = r'message\s+(\w+)\s*{([^}]+)}'
        for match in re.finditer(message_pattern, content, re.MULTILINE | re.DOTALL):
            message_name = match.group(1)
            message_body = match.group(2)
            
            # For the main message, use the proto filename as table name
            table_name = (table_name_base if message_name.lower() == 'root' 
                         else f"{table_name_base}_{message_name.lower()}")
            
            columns = []
            foreign_keys = []
            
            # Add id column as primary key
            columns.append(Column(
                name='id',
                data_type='SERIAL',
                is_nullable=False,
                is_primary_key=True
            ))
            
            # Parse fields
            for line in message_body.split('\n'):
                line = line.strip()
                if not line or line.startswith('//'):
                    continue
                    
                field_match = re.match(
                    r'(repeated|optional|required)?\s*(\w+)\s+(\w+)\s*=\s*(\d+)',
                    line
                )
                if field_match:
                    modifier, field_type, field_name, number = field_match.groups()
                    
                    if modifier == 'repeated':
                        # Create a separate table for repeated fields
                        array_table_name = f"{table_name}_{field_name}"
                        self._handle_repeated_field(
                            array_table_name,
                            field_type,
                            table_name
                        )
                    else:
                        columns.append(Column(
                            name=field_name,
                            data_type=self.TYPE_MAPPING.get(field_type, 'TEXT'),
                            is_nullable=modifier != 'required'
                        ))
            
            self.tables[table_name] = Table(
                name=table_name,
                columns=columns,
                foreign_keys=foreign_keys
            )
            
        return self.tables

    def _handle_repeated_field(self, array_table_name: str, field_type: str, parent_table: str):
        """Create a separate table for repeated fields."""
        columns = [
            Column(name='id', data_type='SERIAL', is_nullable=False, is_primary_key=True),
            Column(name='parent_id', data_type='BIGINT', is_nullable=False),
            Column(
                name='value',
                data_type=self.TYPE_MAPPING.get(field_type, 'TEXT'),
                is_nullable=False
            )
        ]
        
        foreign_keys = [
            f"FOREIGN KEY (parent_id) REFERENCES {parent_table}(id)"
        ]
        
        self.tables[array_table_name] = Table(
            name=array_table_name,
            columns=columns,
            foreign_keys=foreign_keys
        )

class SchemaManager:
    def __init__(self, db_params: Dict[str, str], proto_dir: str, sql_manager: SQLFileManager):
        self.db_params = db_params
        self.proto_dir = proto_dir
        self.sql_manager = sql_manager
        self.conn = self._connect_db()
        self.db_schema = DatabaseSchema(self.conn)

    def _connect_db(self) -> connection:
        """Establish database connection."""
        try:
            return psycopg2.connect(**self.db_params)
        except Exception as e:
            logger.error(f"Failed to connect to database: {e}")
            raise

    def generate_migration(self, new_tables: Dict[str, Table]) -> List[str]:
        """Generate SQL statements for schema migration."""
        current_tables = self.db_schema.current_tables
        migration_statements = []

        for table_name, new_table in new_tables.items():
            if table_name not in current_tables:
                # Create new table
                migration_statements.extend(self._create_table_statements(new_table))
            else:
                # Modify existing table
                current_table = current_tables[table_name]
                migration_statements.extend(
                    self._alter_table_statements(current_table, new_table)
                )

        return migration_statements

    def _create_table_statements(self, table: Table) -> List[str]:
        """Generate SQL statements to create a new table."""
        column_defs = []
        for col in table.columns:
            nullable = "NULL" if col.is_nullable else "NOT NULL"
            pk = "PRIMARY KEY" if col.is_primary_key else ""
            column_defs.append(
                f"{col.name} {col.data_type} {nullable} {pk}".strip()
            )

        statements = [
            f"""
            CREATE TABLE {table.name} (
                {','.join(column_defs)}
            );
            """
        ]

        # Add foreign key constraints
        for fk in table.foreign_keys:
            statements.append(
                f"ALTER TABLE {table.name} ADD CONSTRAINT fk_{table.name} {fk};"
            )

        return statements

    def _alter_table_statements(self, current: Table, new: Table) -> List[str]:
        """Generate SQL statements to modify an existing table."""
        statements = []
        
        # Find columns to add, modify, or remove
        current_cols = {col.name: col for col in current.columns}
        new_cols = {col.name: col for col in new.columns}
        
        # Add new columns
        for col_name, col in new_cols.items():
            if col_name not in current_cols:
                nullable = "NULL" if col.is_nullable else "NOT NULL"
                statements.append(
                    f"ALTER TABLE {current.name} ADD COLUMN {col_name} "
                    f"{col.data_type} {nullable};"
                )
        
        # Modify existing columns
        for col_name, new_col in new_cols.items():
            if col_name in current_cols:
                current_col = current_cols[col_name]
                if (new_col.data_type != current_col.data_type or
                    new_col.is_nullable != current_col.is_nullable):
                    nullable = "NULL" if new_col.is_nullable else "NOT NULL"
                    statements.append(
                        f"ALTER TABLE {current.name} ALTER COLUMN {col_name} "
                        f"TYPE {new_col.data_type} USING {col_name}::{new_col.data_type}, "
                        f"ALTER COLUMN {col_name} SET {nullable};"
                    )
        
        # Add new foreign keys
        current_fks = set(current.foreign_keys)
        new_fks = set(new.foreign_keys)
        for fk in new_fks - current_fks:
            statements.append(
                f"ALTER TABLE {current.name} ADD CONSTRAINT fk_{current.name}_{len(statements)} {fk};"
            )

        return statements

    def process_proto_file(self, proto_file: Path) -> Optional[Path]:
        """Process a single proto file and generate migration."""
        logger.info(f"Processing proto file: {proto_file}")
        
        try:
            # Convert proto to SQL tables
            converter = ProtobufToSQLConverter(str(proto_file))
            new_tables = converter.parse_proto_file(str(proto_file))
            
            # Generate migration statements
            migration_statements = self.generate_migration(new_tables)
            
            # Write migration to SQL file
            if migration_statements:
                proto_name = proto_file.stem
                sql_file = self.sql_manager.write_migration(proto_name, migration_statements)
                logger.info(f"Migration file created: {sql_file}")
                
                # Execute migration
                logger.info(f"Executing migration for {proto_file}")
                with self.conn.cursor() as cur:
                    for statement in migration_statements:
                        logger.info(f"Executing: {statement}")
                        cur.execute(statement)
                self.conn.commit()
                logger.info(f"Successfully migrated schema for {proto_file}")
                return sql_file
            else:
                logger.info(f"No schema changes needed for {proto_file}")
                return None

        except Exception as e:
            logger.error(f"Error processing {proto_file}: {e}")
            self.conn.rollback()
            raise

def main():
    # Get the script's directory and construct paths
    script_dir = Path(__file__).parent
    project_root = script_dir.parent
    proto_dir = project_root / 'proto'
    sql_dir = project_root / 'sql'

    try:
        # Load database configuration
        db_params = ConfigManager.load_db_config()
        
        # Initialize SQL file manager
        sql_manager = SQLFileManager(sql_dir)
        
        # Initialize schema manager
        schema_manager = SchemaManager(db_params, str(proto_dir), sql_manager)
        
        # Process each proto file
        proto_files = list(proto_dir.glob('*.proto'))
        if not proto_files:
            logger.warning(f"No .proto files found in {proto_dir}")
            return

        migration_files = []
        for proto_file in proto_files:
            sql_file = schema_manager.process_proto_file(proto_file)
            if sql_file:
                migration_files.append(sql_file)

        if migration_files:
            logger.info("\nMigration files created:")
            for file in migration_files:
                logger.info(f"- {file}")
        else:
            logger.info("No schema changes were necessary.")

    except Exception as e:
        logger.error(f"Error during schema migration: {e}")
        raise
    finally:
        schema_manager.conn.close()

if __name__ == "__main__":
    main()