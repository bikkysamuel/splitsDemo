# Server data access is hand-written SQL via sqlc + pgx, with no ORM

Queries are plain SQL files compiled into type-safe Go by sqlc and run through pgx v5. Migrations are forward-only goose SQL files embedded in the binary. We rejected ORMs (GORM, ent) because money queries (Balances over 10k+ Expenses, Pending/accepted filters) need exact, reviewable SQL and predictable transactions. Exchange rates are Postgres `NUMERIC`, handled in Go as `math/big.Rat`, so no decimal library is needed.
