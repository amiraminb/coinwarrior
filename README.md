# coinwarrior

Local-first CLI tool for tracking personal finances.

## Current Features

- Interactive transaction entry: `coinw add`
- Interactive transaction editing: `coinw edit`
- Interactive transaction deletion: `coinw delete`
- Interactive account management: `coinw account`
- Monthly budgets with rollover: `coinw budget`
- List transactions: `coinw list [range]`
- Range report (balances + category): `coinw report <range> [--details]`

## Installation

### Pre-built binary (recommended)

Download the latest release for your platform from the [Releases page](https://github.com/amiraminb/coinwarrior/releases), extract, and place `coinw` somewhere on your `PATH`.

One-liner for macOS/Linux (replace `VERSION` with the latest tag):

```bash
VERSION=0.1.0
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/aarch64/arm64/;s/x86_64/x86_64/')
curl -sSL "https://github.com/amiraminb/coinwarrior/releases/download/v${VERSION}/coinwarrior_${VERSION}_${OS}_${ARCH}.tar.gz" \
  | tar xz -C /tmp \
  && mv /tmp/coinw ~/.local/bin/coinw
```

### From source with Go

```bash
go install github.com/amiraminb/coinwarrior@latest
```

### From source with make

Builds and copies `coinw` to `~/.local/bin`:

```bash
make build
```

If `~/.local/bin` is not on your `PATH`, either add it or run with the full path: `~/.local/bin/coinw <command>`.

## Quick Start

- Initialize data files:

```bash
coinw init
```

- Add at least one account:

```bash
coinw account
```

- Add a transaction:

```bash
coinw add
```

- List transactions:

```bash
coinw list
coinw list <range>
```

- Generate a report:

```bash
coinw report <range>
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
