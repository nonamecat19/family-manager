# MCP servers

Project-scoped, in `.mcp.json`. Claude Code asks for approval on first use per server.
Everything else this repo needs — files, search, git, builds — is already covered by native
tools, so no MCP server is added for it.

| server | why it earns its place |
|---|---|
| `postgres` | live schema of the dev database: verify a migration actually created what the graph claims, check indexes/constraints before writing sqlc queries, explain a slow query. The knowledge graph knows what the SQL *files* say; this knows what the database *is*. |
| `context7` | version-correct docs for Expo, Turborepo, pgx, grpc-go, sqlc. Prevents API drift from training-data recall. |

## postgres

Runs against the local compose database only.

```sh
just up          # postgres must be running
uvx --help       # requires uv: https://docs.astral.sh/uv/
```

`--access-mode=restricted` keeps it read-only and caps expensive queries. **Never point
`DATABASE_URI` at a production database** — an agent with write access to prod is an
irreversible edge with no gate on it. Migrations are applied through `just`/`make migrate` and
the human gate, never through this server.

Credentials come from the same env vars as `docker-compose.yml`; nothing is committed.

## context7

No API key needed for basic use (rate-limited). A global `context7-mcp` skill may already be
available; the server here makes it work without it.

## Deliberately not added

- **filesystem / git / fetch servers** — native Read/Edit/Grep/Bash/WebFetch already do this,
  and duplicate tools cost context on every turn.
- **a graph database server** — the knowledge graph is a JSON file queried with `just impact`
  and `jq`. Adding Neo4j would buy nothing at this repo's size.
