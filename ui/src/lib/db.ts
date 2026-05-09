import Database from 'better-sqlite3'

let db: Database.Database | null = null

export function getDb(): Database.Database {
  if (!db) {
    const path = process.env.DATABASE_PATH || './../data/sniffy.db'
    db = new Database(path, { readonly: true })
  }
  return db
}

export function closeDb(): void {
  if (db) {
    db.close()
    db = null
  }
}
