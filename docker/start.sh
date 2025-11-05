#!/bin/bash
set -e

# Default values
POSTGRES_USER="${POSTGRES_USER:-cnap}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-cnap}"
POSTGRES_DB="${POSTGRES_DB:-cnap}"
PGDATA="${PGDATA:-/var/lib/postgresql/data}"

echo "Starting unified container..."

# Initialize PostgreSQL if not already initialized
if [ ! -s "$PGDATA/PG_VERSION" ]; then
    echo "Initializing PostgreSQL database..."

    # Clean up any partial initialization
    rm -rf "$PGDATA"/*

    su-exec postgres initdb -D "$PGDATA"

    # Configure PostgreSQL
    echo "host all all 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"
    echo "listen_addresses='*'" >> "$PGDATA/postgresql.conf"
fi

# Start PostgreSQL in background
echo "Starting PostgreSQL..."
su-exec postgres postgres -D "$PGDATA" &
POSTGRES_PID=$!

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if su-exec postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" > /dev/null 2>&1; then
        echo "PostgreSQL is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "PostgreSQL failed to start"
        exit 1
    fi
    sleep 1
done

# Create user and database if they don't exist
su-exec postgres psql -v ON_ERROR_STOP=1 <<-EOSQL
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$POSTGRES_USER') THEN
            CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD';
        END IF;
    END
    \$\$;

    SELECT 'CREATE DATABASE $POSTGRES_DB OWNER $POSTGRES_USER'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$POSTGRES_DB')\gexec

    GRANT ALL PRIVILEGES ON DATABASE $POSTGRES_DB TO $POSTGRES_USER;
EOSQL

echo "Starting CNAP application..."
# Start the application in foreground
exec /app/cnap start
