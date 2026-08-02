# coinwarrior

Local-first CLI tool for tracking personal finances.

## Current Features

- Interactive transaction entry: `coinw add`
- Interactive CSV import: `coinw import <file.csv>`
- Duplicate warning on `add` and `import` when a transaction matches an existing
  one's date, category, amount, and type
- Interactive transaction editing: `coinw edit` (category is chosen from a list)
- Interactive transaction deletion: `coinw delete`
- Interactive account management: `coinw account`
- Monthly budgets with rollover: `coinw budget`
- Recurring transactions with monthly cadence: `coinw recurring`
- Account balances report: `coinw report account`
- Range overview report: `coinw report overview [category] <range>`
- Transactions report: `coinw report transactions [category] <range>`

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

If the transaction you are entering matches an existing one on date, category,
amount, and type, `add` asks for confirmation before saving so an accidental
double entry is caught. `import` does the same on each row, including against
rows saved earlier in the same run. Transfers are never flagged, since they all
share the `Transfer` category and legitimately repeat. Currency is not part of
the match, so `20.00 CAD` and `20.00 USD` on the same date and category both
trigger the warning.

- List transactions in a range:

```bash
coinw report transactions <range>
```

- List transactions for a single category, with an income/expense summary:

```bash
coinw report transactions <category> <range>
```

The category is matched case-insensitively against your saved categories, so
`groceries` and `Groceries` both work. An unknown name fails with the list of
valid categories rather than printing an empty table. Transfers are excluded
from a category-filtered listing, matching `coinw report overview`, so the table
and the summary below it always agree.

Each row is coloured by its category, so transactions can be grouped by eye.
Colours follow the category's position in your saved list and so stay the same
between runs. Only five hues are used, since that is the most a terminal can show
while staying distinguishable for colour-blind and normal vision on both light
and dark backgrounds; a sixth category reuses a hue, and the category name beside
it tells them apart.

- Generate an overview for a range (per-category totals, an income/expense
  summary, and the month's budget when the range is exactly one calendar month):

```bash
coinw report overview <range>
```

A range spanning more than one month also gets a per-month income/expense bar
chart. Bars scale to the largest value in the range, and a month the range only
partly covers is labelled with the days it covers:

```
Monthly Income / Expense (CAD)

 MONTH              INCOME                        EXPENSE
 2026-01 (15-31)    ███████████████████ 5,000.00  ████████████ 3,240.50
 2026-02            ███████████████████ 5,000.00  ████████████████████████ 6,100.00
 2026-03 (01-20)    █████████ 2,500.00            ███████ 1,890.00
```

Below the chart, each month gets its own category totals and income/expense
summary, so you can see where that month's money went. A partial month counts
only the days the range covers, and the `% EXP` share is recomputed within each
month rather than carried over from the whole range.

Every saved category is listed, including the ones with no activity, so each
month has the same set of rows and a category you spent nothing on is visible
rather than missing.

Each per-month table also compares itself to the month immediately before it. Red
`▲` means the month moved for the worse (more spent, or less earned) and green `▼`
means it improved. A category with no figure last month is compared against zero:

```
 CATEGORY              CUR    TOTAL      TXNS   % EXP     VS PREV
 Groceries             CAD    -150.00    1      100.0%    ▲ 50.00
 Income                CAD    5,500.00   1      0.0%      ▼ 1,000.00
 Dining                CAD    0.00       0      -         =
```

Pass a category to narrow every section, including the bars, to just that
category:

```bash
coinw report overview <category> <range>
```

- Show account balances:

```bash
coinw report account
```

## Supported Report Ranges

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
coinw report transactions 2026-04-01..2026-04-30
coinw report transactions Groceries 2026-04-01..2026-04-30
coinw report overview 2026-04-01..2026-04-30
```
