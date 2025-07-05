import { defineConfig } from 'drizzle-kit'
import 'dotenv/config'

export default defineConfig({
  schema: './src/**/*.schema.ts',
  dialect: 'postgresql',
  dbCredentials: {
    // biome-ignore lint/style/noNonNullAssertion: env
    url: process.env.DATABASE_URL!,
  },
})
