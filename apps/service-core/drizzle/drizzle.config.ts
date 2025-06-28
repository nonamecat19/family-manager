/** biome-ignore-all lint/style/noNonNullAssertion: env */
import { defineConfig } from 'drizzle-kit'
import 'dotenv/config'
import * as process from 'node:process'

export default defineConfig({
  schema: ['./src/schema.ts'],
  dialect: 'postgresql',
  dbCredentials: {
    user: process.env.DB_USER!,
    host: process.env.DB_HOST!,
    database: process.env.DB_NAME!,
    password: process.env.DB_PASSWORD!,
    port: parseInt(process.env.DB_PORT!),
    ssl: false,
  },
})
