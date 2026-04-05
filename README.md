# zenmoney-go

Go client library and CLI for the [ZenMoney API](https://github.com/zenmoney/ZenPlugins/wiki/ZenMoney-API).

ZenMoney automatically imports transactions from Russian banks (Tinkoff, Sber, Alfa, etc.) via SMS/push parsing. This library gives you programmatic access to that data.

## Installation

```bash
go install github.com/miilg/zenmoney-go/cmd/zm@latest
```

Or build from source:

```bash
git clone https://github.com/miilg/zenmoney-go.git
cd zenmoney-go
go build -o zm ./cmd/zm/
# Move to a directory in your PATH:
cp zm /usr/local/bin/   # or ~/go/bin/, ~/.local/bin/, etc.
```

## Quick Start

```bash
# 1. Get a token at https://zerro.app/token
# 2. Save it
zm auth --token <YOUR_TOKEN>

# 3. Pull all data from ZenMoney
zm sync

# 4. Explore
zm accounts
zm tx
zm summary
```

## Claude Code Skill

This repo includes a [Claude Code skill](https://docs.anthropic.com/en/docs/claude-code/skills) that lets Claude answer finance questions using `zm` as a tool.

### Setup

Make sure `zm` is installed and in your PATH, then choose one option:

**Option A — Personal skill (available in all projects):**

```bash
cp -r .claude/skills/zenmoney ~/.claude/skills/zenmoney
```

**Option B — Add as a directory:**

```bash
# When starting Claude Code, add this repo:
claude --add-dir /path/to/zenmoney-go
```

**Option C — Project skill (if working inside this repo):**

The skill is already at `.claude/skills/zenmoney/` — it works automatically.

### Usage

Once the skill is active, just ask Claude about your finances in natural language:

```
> How much did I spend on food this month?
> Show my account balances
> Compare my income in January vs February
> What's my net worth?
```

Or invoke directly: `/zenmoney show my last 50 transactions`

## CLI Commands

### `zm auth`

Save an access token. Get one at [zerro.app/token](https://zerro.app/token) or via OAuth2 (see Library section).

```bash
zm auth --token <TOKEN>
```

### `zm sync`

Sync data from ZenMoney. First run pulls everything; subsequent runs are incremental.

```bash
zm sync          # incremental
zm sync --full   # force full re-sync
```

### `zm accounts`

List accounts with balances.

```bash
zm accounts          # active accounts only
zm accounts --all    # include archived
```

```
TITLE                TYPE      BALANCE      CURRENCY
Black                ccard     18 264.80    RUB
Основной аккаунт     checking  239 478.70   RUB
SafePal              checking  156          USDT
```

### `zm transactions`

List transactions with filters. Alias: `zm tx`.

```bash
zm tx                              # last 30 transactions
zm tx --limit 100                  # last 100
zm tx --from 2026-03-01            # from date
zm tx --from 2026-03-01 --to 2026-03-31
zm tx --account "Tinkoff"          # filter by account name (substring)
zm tx --tag "Кафе"                 # filter by category (substring)
```

```
DATE        AMOUNT         ACCOUNT  CATEGORY          PAYEE
2026-04-05  -2 650 руб.    Black    Кафе и рестораны   Nothing Fancy
2026-04-04  -670 руб.      Black    Транспорт          Такси
```

### `zm categories`

List all categories as a tree. Alias: `zm cats`.

### `zm summary`

Monthly income/expense breakdown + net worth.

```bash
zm summary               # last 3 months
zm summary --months 12   # last year
```

```
MONTH    INCOME        EXPENSE       NET
2026-01  4 054 528     3 832 372.70  +222 155.30    RUB
2026-02  7 441 392     6 763 659.28  +677 732.72    RUB
2026-03  56 917        581 298.95    -524 381.95    RUB

Net worth: 483 752.49 RUB
```

## Library Usage

### Basic sync

```go
package main

import (
    "fmt"
    zm "github.com/miilg/zenmoney-go"
)

func main() {
    client := zm.NewClient("your-access-token")

    // Initial full sync
    data, err := client.FetchAll()
    if err != nil {
        panic(err)
    }
    fmt.Printf("%d transactions, %d accounts\n",
        len(data.Transaction), len(data.Account))

    // Incremental sync — pass the serverTimestamp from the previous response
    updates, _ := client.Diff(&zm.DiffRequest{
        ServerTimestamp: data.ServerTimestamp,
    })
    fmt.Printf("%d updated transactions\n", len(updates.Transaction))
}
```

### Push changes

```go
client.Push(serverTimestamp,
    zm.WithTransactions([]zm.Transaction{myTx}),
    zm.WithAccounts([]zm.Account{myAccount}),
)
```

### Suggest categories

```go
payee := "McDonalds"
sug, _ := client.Suggest(&zm.SuggestRequest{Payee: &payee})
fmt.Println(sug.Tag) // suggested category IDs
```

### OAuth2 flow

```go
cfg := &zm.OAuthConfig{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    RedirectURI:  "http://localhost:8080/callback",
}

// 1. Redirect user to:
url := cfg.AuthCodeURL()

// 2. Exchange the code from callback:
token, _ := cfg.Exchange(code)

// 3. Refresh when expired:
newToken, _ := cfg.Refresh(token.RefreshToken)
```

### Local storage

```go
store, _ := zm.NewStore() // ~/.config/zenmoney/

// Save/load token
store.SaveToken(token)
token, _ := store.LoadToken()

// Save/load synced data with incremental merge
state, _ := store.LoadSync()
state.MergeSync(diffResponse)
store.SaveSync(state)
```

## Data Model

All [ZenMoney API entities](https://github.com/zenmoney/ZenPlugins/wiki/ZenMoney-API) are mapped to Go structs:

| Struct | Description | CRUD |
|--------|-------------|------|
| `Transaction` | Income/expense/transfer with multi-currency support | read/write |
| `Account` | Bank accounts, cards, cash, loans, deposits | read/write |
| `Tag` | Categories with one level of nesting | read/write |
| `Merchant` | Payee entities | read/write |
| `Budget` | Monthly budgets per category | read/write |
| `Reminder` | Recurring transaction templates | read/write |
| `ReminderMarker` | Scheduled transaction instances | read/write |
| `Instrument` | Currencies with exchange rates | read-only |
| `Company` | Banks and financial institutions | read-only |
| `User` | Account owners (family support) | read-only |

## How Sync Works

ZenMoney uses differential sync via a single `POST /v8/diff/` endpoint:

1. Client sends `serverTimestamp` from the last sync (0 for initial)
2. Server returns all entities changed since that timestamp
3. Client can also send local changes in the same request
4. Server returns a new `serverTimestamp` to use next time

The `SyncState.MergeSync()` method handles merging incremental updates and deletions into the local state.

## Data Storage

Token and synced data are stored in `~/.config/zenmoney/`:

```
~/.config/zenmoney/
├── token.json   # OAuth access/refresh token
└── data.json    # All synced entities + serverTimestamp
```

## Zero Dependencies

The library and CLI use only the Go standard library.
