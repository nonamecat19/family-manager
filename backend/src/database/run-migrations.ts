import { readdir, readFile } from 'fs/promises';
import { join } from 'path';
import postgres from 'postgres';

interface JournalEntry {
  idx: number;
  version: string;
  when: number;
  tag: string;
  breakpoints: boolean;
}

interface Journal {
  version: string;
  dialect: string;
  entries: JournalEntry[];
}

async function runMigrations() {
  const databaseUrl = process.env.DATABASE_URL;
  
  if (!databaseUrl) {
    throw new Error('DATABASE_URL environment variable is not set');
  }

  console.log('🔄 Connecting to database...');
  const sql = postgres(databaseUrl, {
    max: 1,
    idle_timeout: 20,
    connect_timeout: 10,
  });

  try {
    // Read journal file to get migration order and metadata
    const journalPath = join(__dirname, 'migrations', 'meta', '_journal.json');
    let journal: Journal;
    
    try {
      const journalContent = await readFile(journalPath, 'utf-8');
      journal = JSON.parse(journalContent);
    } catch (error) {
      console.log('⚠️  Journal file not found, scanning migrations directory...');
      // Fallback: read migrations directory
      const migrationsPath = join(__dirname, 'migrations');
      const files = await readdir(migrationsPath);
      const sqlFiles = files
        .filter(file => file.endsWith('.sql'))
        .sort();
      
      journal = {
        version: '7',
        dialect: 'postgresql',
        entries: sqlFiles.map((file, idx) => ({
          idx,
          version: '7',
          when: Date.now(),
          tag: file.replace('.sql', ''),
          breakpoints: true,
        })),
      };
    }

    if (journal.entries.length === 0) {
      console.log('✅ No migrations found');
      return;
    }

    console.log(`📦 Found ${journal.entries.length} migration(s)`);

    // Check if migrations table exists, create if not (drizzle format)
    await sql`
      CREATE TABLE IF NOT EXISTS __drizzle_migrations (
        id SERIAL PRIMARY KEY,
        hash text NOT NULL,
        created_at bigint
      )
    `;

    // Get already applied migrations
    const appliedMigrations = await sql`
      SELECT hash FROM __drizzle_migrations
    `;
    const appliedHashes = new Set(appliedMigrations.map((m: any) => m.hash));

    // Run each migration in order
    for (const entry of journal.entries) {
      const sqlFile = `${entry.tag}.sql`;
      const filePath = join(__dirname, 'migrations', sqlFile);
      
      // Use tag as hash (drizzle uses tag for tracking)
      const hash = entry.tag;

      if (appliedHashes.has(hash)) {
        console.log(`⏭️  Skipping ${sqlFile} (already applied)`);
        continue;
      }

      console.log(`🔄 Running migration: ${sqlFile}`);
      
      try {
        const content = await readFile(filePath, 'utf-8');
        
        // Execute migration in a transaction
        await sql.begin(async (tx) => {
          // Split by semicolon and execute each statement
          const statements = content
            .split(';')
            .map(s => s.trim())
            .filter(s => s.length > 0 && !s.startsWith('--'));

          for (const statement of statements) {
            if (statement.trim()) {
              await tx.unsafe(statement);
            }
          }

          // Record migration (drizzle format)
          await tx`
            INSERT INTO __drizzle_migrations (hash, created_at)
            VALUES (${hash}, ${entry.when || Date.now()})
          `;
        });

        console.log(`✅ Applied migration: ${sqlFile}`);
      } catch (error) {
        console.error(`❌ Failed to apply migration ${sqlFile}:`, error);
        throw error;
      }
    }

    console.log('✅ All migrations completed successfully');
  } catch (error) {
    console.error('❌ Migration failed:', error);
    throw error;
  } finally {
    await sql.end();
  }
}

// Run migrations if this file is executed directly
if (require.main === module) {
  runMigrations()
    .then(() => {
      console.log('🚀 Migrations finished');
      process.exit(0);
    })
    .catch((error) => {
      console.error('💥 Migration error:', error);
      process.exit(1);
    });
}

export { runMigrations };

