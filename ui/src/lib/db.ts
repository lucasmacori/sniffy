import Database from 'better-sqlite3'

let db: Database.Database | null = null

function getDbPath(): string {
  return process.env.DATABASE_PATH || './../data/sniffy.db'
}

export function getDb(): Database.Database {
  if (!db) {
    db = new Database(getDbPath())
  }
  return db
}

export function getRwDb(): Database.Database {
  return new Database(getDbPath())
}

export function closeDb(): void {
  if (db) {
    db.close()
    db = null
  }
}
