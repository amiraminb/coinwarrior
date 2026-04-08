# coinwarrior

Local-first CLI tool for tracking personal finances.

## Current Features

- Interactive transaction entry: `coinw add`
- Interactive transaction editing: `coinw edit`
- Interactive transaction deletion: `coinw delete`
- Interactive account setup: `coinw account add`
- Update account balance: `coinw account update`
- List transactions: `coinw list [range]`
- Range report (balances + category): `coinw report <range> [--details]`

## Quick Start

Build the binary first (this installs `coinw` to `~/.local/bin`):

```bash
make build
```

If `~/.local/bin` is not on your `PATH`, run with the full path:

```bash
~/.local/bin/coinw <command>
```

1. Initialize data files:

```bash
coinw init
```

1. Add at least one account:

```bash
coinw account add
```

1. Add a transaction:

```bash
coinw add
```

1. List transactions:

```bash
coinw list
coinw list month
```

1. Generate a report:

```bash
coinw report month
```

## Supported List/Report Ranges

- `today`
- `yesterday`
- `week`
- `lastweek`
- `month`
- `lastmonth`
- `year`
- `lastyear`
- `<YYYY-MM-DD..YYYY-MM-DD>`

Example:

```bash
coinw list 2026-04-01..2026-04-30
coinw report 2026-04-01..2026-04-30
```
