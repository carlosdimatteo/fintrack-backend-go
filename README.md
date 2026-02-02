# Fintrack Backend

Go API for the Fintrack app: track income, expenses, budgets, investments, debts, and transfers; keep a Google Sheet in sync; and run monthly reconciliation and net-worth views.

## What it does

- **Expenses & income** — Record and list with categories
- **Budgets** — Per-category budgets and history.
- **Investments** — Deposit/withdrawal by account; expected vs real capital.
- **Debts** — Lend/borrow tracking; expense-debt and repayments.
- **Transfers** — Between accounts.
- **Accounting** — Set real balances at month-end; expected vs real and discrepancy per account.
- **Goals & net worth** — Yearly goals, net-worth snapshots, dashboard (income, expenses, savings, progress).
- **Exchange rate** - exchange rate functionality 
- **Google Sheet** — Writes rows to a connected sheet.


## How it’s built

- **Go** — Single binary; just `net/http` + [gorilla/mux](https://github.com/gorilla/mux).
- **Postgres** — [pgx](https://github.com/jackc/pgx) for all persistence.
- **Google Sheets** — Adapter in `adapters/google/` for appending/updating cells.
- **Layout** — `api/` (handlers + routes), `adapters/`, `helpers/`, `types/`.

## How to run

**Requirements:** Go 1.24+, Postgres, and for sheet sync: a Google project.

```bash
# Install deps
go mod download

# Run (loads .env if present; default port 3001)
go run fintrack.go
```

Or with [Air](https://github.com/air-verse/air) for reload on change (see `.air.toml`):

```bash
air
```

**Env (summary):** `DATABASE_URL`, `API_KEY`, `ALLOWED_ORIGINS`; optional: `EXCHANGERATE_API_KEY`, `GO_ENV=dev` (skips sheet writes and relaxes middlewares).


Tests: `go test ./...`
