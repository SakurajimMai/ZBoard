#!/usr/bin/env sh
set -eu

DB_PATH="${ZBOARD_DB_PATH:-/data/zboard.sqlite}"
BACKUP_DIR="${ZBOARD_BACKUP_DIR:-/backups}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
BACKUP_FILE="${BACKUP_DIR}/zboard-${TIMESTAMP}.sqlite"

mkdir -p "${BACKUP_DIR}"

if [ ! -f "${DB_PATH}" ]; then
  echo "database file not found: ${DB_PATH}" >&2
  exit 1
fi

cp "${DB_PATH}" "${BACKUP_FILE}"
gzip -f "${BACKUP_FILE}"

find "${BACKUP_DIR}" -name 'zboard-*.sqlite.gz' -type f -mtime +30 -delete

echo "${BACKUP_FILE}.gz"
