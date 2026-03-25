# jiff

Intent-aware JSON diff CLI written in Go.

## What it does

- Compares JSON by meaning, not text formatting.
- Ignores key order in objects.
- Supports recursive ignored fields.
- Handles arrays with index mode, unordered mode, or key-based matching.
- Produces summary, verbose, raw JSON, or full colorized diff output.

## Build

Requirements: Go 1.22+

```bash
go build -o jiff .
```

## Usage

```bash
jiff <file1> <file2> [--ignore fields] [--match key] [--unordered] [--summary|--verbose|--raw|--full]
```

## Flags

- `--ignore id,updatedAt,createdAt`
	- Comma-separated fields to remove recursively before diffing.
- `--match id`
	- Matches objects inside arrays by key value.
- `--unordered`
	- Treats arrays as unordered multisets.
- `--summary`
	- Minimal human-readable output (default).
- `--verbose`
	- Human-readable output including removed values.
- `--raw`
	- Machine-readable JSON output.
- `--full`
	- Classic full line diff with colorized additions/removals.

## Examples

Summary mode (default):

```bash
jiff a.json b.json --ignore updatedAt,id --match id
```

Verbose mode:

```bash
jiff a.json b.json --verbose
```

Raw JSON output:

```bash
jiff a.json b.json --raw
```

Full colorized diff output:

```bash
jiff a.json b.json --full
```

Unordered array compare:

```bash
jiff a.json b.json --unordered
```
