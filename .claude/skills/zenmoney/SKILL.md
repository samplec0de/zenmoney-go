---
name: zenmoney
description: Query personal finances via ZenMoney API. Use when the user asks about money, finances, accounts, balances, transactions, spending, income, expenses, budget, net worth, savings, or cash flow.
argument-hint: "[question or zm command]"
allowed-tools: Bash(zm *)
---

# ZenMoney Finance CLI

You have access to `zm` — a CLI that syncs and queries financial data from ZenMoney.

ZenMoney connects to Russian banks (Tinkoff, Sber, Alfa, etc.) and imports transactions automatically via SMS/push parsing on the user's phone. The `zm` CLI reads that data through the ZenMoney API.

## Commands

```
zm sync                   # Pull latest data (incremental)
zm sync --full            # Force full re-sync from scratch
zm accounts               # List accounts with balances and currency
zm accounts --all         # Include archived accounts
zm tx                     # Last 30 transactions
zm tx --limit N           # Last N transactions (0 = unlimited)
zm tx --from YYYY-MM-DD   # Start date filter
zm tx --to YYYY-MM-DD     # End date filter
zm tx --account "text"    # Filter by account name (case-insensitive substring)
zm tx --tag "text"        # Filter by category name (case-insensitive substring)
zm cats                   # List categories as a tree
zm summary                # Monthly income/expense/net + net worth (last 3 months)
zm summary --months N     # Last N months
```

Flags combine freely: `zm tx --from 2026-01-01 --to 2026-01-31 --tag "Food" --limit 500`

## Workflow

1. **Sync first** — run `zm sync` before answering if you haven't synced this session.
2. **Compute** — the CLI outputs raw data. Sum totals, compute averages, compare periods, spot trends yourself.
3. **Combine commands** — a question like "how much did I spend on food in March?" needs `zm tx --tag "..." --from ... --to ... --limit 0`, then sum the outcome amounts.
4. **Transfers** — transactions with both income > 0 and outcome > 0 between different accounts are transfers, not income/expense. `zm summary` already handles this.
5. **Multi-currency** — accounts can be in different currencies (RUB, USD, USDT, etc.). The CURRENCY column in `zm accounts` shows which. `zm summary` converts to the user's base currency.

## Error handling

| Error | Fix |
|-------|-----|
| "not authenticated" | User needs a token: `zm auth --token <TOKEN>` (get at https://zerro.app/token) |
| "no data" | Run `zm sync` first |
| Empty results | Widen filters — check date range, try without --tag/--account |

## Data location

- Config & data: `~/.config/zenmoney/`
- Source: the zenmoney-go repository
