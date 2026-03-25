# jiff — Spec

## 1. Overview

`jiff` is a CLI tool for comparing JSON files with a focus on **meaningful, intent-aware, human-readable diffs**.

Unlike traditional tools, it:

* prioritizes semantic differences over structural noise
* handles arrays intelligently
* allows user-defined intent (ignore fields, match keys)
* outputs clean, scannable summaries

---

## 2. Goals

### Primary Goals

* Show only **meaningful changes**
* Provide **clean, minimal output**
* Support **real-world workflows** (APIs, configs, data)

### Non-Goals

* Not a line-by-line diff replacement
* Not a full JSON transformation tool (like jq)

---

## 3. CLI Interface

### Basic Usage

```
jiff <file1> <file2>
```

### Core Flags

#### Ignore fields

```
--ignore <fields>
```

Comma-separated list:

```
--ignore id,updatedAt,createdAt
```

---

#### Match key for arrays

```
--match <key>
```

Used to match objects inside arrays:

```
--match id
```

---

#### Treat arrays as unordered

```
--unordered
```

---

#### Output modes

```
--summary
--verbose
--raw
```

* summary (default): minimal human-readable
* verbose: includes full paths and values
* raw: structured JSON diff output

---

## 4. Output Format

### Default (Summary Mode)

```
Changed (2):
  user.name: "Alice" → "Bob"

Added (1):
  user.age: 25

Removed (1):
  user.nickname

Unchanged (order only):
  users
```

---

### Verbose Mode

Includes:

* full JSON paths
* before/after values

---

### Raw Mode

Machine-readable JSON output:

```
{
  "changed": [...],
  "added": [...],
  "removed": [...]
}
```

---

## 5. Core Features

### 5.1 Semantic Diffing

* ignore formatting
* ignore key ordering
* compare actual values

---

### 5.2 Intelligent Array Handling

#### Without --match

* fallback to index-based diff

#### With --match

* treat array as collection of objects
* match elements by key

Example:

Input A:

```
[{"id":1,"name":"A"},{"id":2,"name":"B"}]
```

Input B:

```
[{"id":2,"name":"B"},{"id":1,"name":"A"}]
```

Output:

```
Unchanged (order only):
  root
```

---

### 5.3 Field Ignoring

* removes specified fields before diff
* applies recursively

---

### 5.4 Change Categorization

All diffs categorized into:

* Changed
* Added
* Removed
* Reordered

---

## 6. Internal Architecture

### Step 1: Parse

* Load JSON
* Validate structure

### Step 2: Normalize

* Remove ignored fields
* Normalize key ordering

### Step 3: Diff Engine

* Recursive comparison
* Special handling for arrays

### Step 4: Classification

* categorize differences

### Step 5: Output Formatter

* format based on mode

---

## 7. Edge Cases

* Null vs undefined
* Type changes (string → number)
* Deeply nested arrays
* Mixed-type arrays

---

## 8. MVP Scope (v1)

Must have:

* file comparison
* ignore fields
* match key for arrays
* summary output

Nice to have:

* unordered arrays flag
* verbose mode

Not in v1:

* patch generation
* YAML support
* config files

---

## 9. Future Enhancements

* Config file support (.jiffrc)
* YAML support
* Git integration
* Interactive CLI (expand/collapse)
* Web UI

---

## 10. Success Criteria

* Faster to understand diff than existing tools
* Less noise in output
* Works well for API responses

---

## 11. Example Command

```
jiff a.json b.json \
  --ignore updatedAt,id \
  --match id \
  --summary
```

---

End of Spec
