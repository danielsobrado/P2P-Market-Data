# Protobuf to PostgreSQL Schema Manager

A [utility](https://github.com/danielsobrado/p2p_market_data/blob/main/scripts/proto_to_sql.py) for automatically generating and managing PostgreSQL database schemas from Protocol Buffer definitions. 

This tool monitors your proto files and automatically creates or updates database tables to match your service definitions, maintaining consistency between your protobuf messages and database schema.

## 🌟 Features

- **Automatic Schema Generation**: Converts Protocol Buffer messages to PostgreSQL tables
- **Smart Schema Migration**: Only generates necessary changes to existing schemas
- **Transaction Safety**: All migrations are wrapped in transactions for safety
- **Version Control Friendly**: Generates timestamped SQL migration files
- **Comprehensive Logging**: Detailed logging of all operations
- **Type Mapping**: Intelligent mapping between protobuf and PostgreSQL types
- **Relationship Handling**: Properly handles repeated fields with foreign key relationships

## 📁 Project Structure

```
p2p_market_data/
├── scripts/
│   └── proto_to_sql.py
├── proto/
│   ├── dividends.proto
│   └── splits.proto
├── sql/
│   ├── 20241109_123456_dividends_migration.sql
│   └── 20241109_123457_splits_migration.sql
└── config/
    └── db_config.yaml
```

## ⚙️ Configuration

Create a `db_config.yaml` file in the `config` directory:

```yaml
database:
  name: your_database
  user: your_username
  password: your_password
  host: localhost
  port: 5432

migrations:
  backup: true
  dry_run: false
```

## 🚀 Installation

1. Clone the repository:
```bash
git clone https://github.com/danielsobrado/p2p_market_data
cd p2p_market_data
```

2. Install dependencies:
```bash
pip install -r requirements.txt
```

## 📦 Dependencies

- Python 3.8+
- psycopg2-binary
- pyyaml
- typing-extensions

## 🔧 Usage

1. Place your `.proto` files in the `proto` directory

2. Configure your database connection in `config/db_config.yaml`

3. Run the migration script:
```bash
python scripts/proto_to_sql.py
```

## 🗃 Type Mappings

| Protobuf Type | PostgreSQL Type |
|---------------|----------------|
| double        | DOUBLE PRECISION |
| float         | REAL |
| int32         | INTEGER |
| int64         | BIGINT |
| uint32        | INTEGER |
| uint64        | BIGINT |
| sint32        | INTEGER |
| sint64        | BIGINT |
| fixed32       | INTEGER |
| fixed64       | BIGINT |
| sfixed32      | INTEGER |
| sfixed64      | BIGINT |
| bool          | BOOLEAN |
| string        | TEXT |
| bytes         | BYTEA |
| timestamp     | TIMESTAMP |

## 🔍 Migration Process

1. **Discovery**: Scans the `proto` directory for `.proto` files
2. **Analysis**: Compares current database schema with proto definitions
3. **Generation**: Creates SQL migration files in the `sql` directory
4. **Execution**: Applies migrations to the database
5. **Logging**: Records all actions and results

## ⚠️ Important Notes

- Always backup your database before running migrations
- The tool uses transactions to ensure safe migrations
- Each migration is saved as a separate SQL file for version control
- Table names are derived from proto file names
- Repeated fields create separate tables with foreign key relationships

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🐛 Known Issues

- Complex nested messages might require manual review of generated schemas
- Some protobuf types might need custom mapping depending on your use case

## 📮 Support

For support, please open an issue in the repository or contact the maintainers.