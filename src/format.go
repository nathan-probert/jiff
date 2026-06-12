package main

import (
    "encoding/json"
    "fmt"
    "strings"
)

func formatResult(result DiffResult, mode outputMode) (string, error) {
    switch mode {
    case modeRaw:
        return formatRaw(result)
    case modeSummary:
        return formatText(result)
    default:
        return formatFull(result)
    }
}

func formatRaw(result DiffResult) (string, error) {
    changed := result.Changed
    if changed == nil {
        changed = []ChangedItem{}
    }
    added := result.Added
    if added == nil {
        added = []AddedItem{}
    }
    removed := result.Removed
    if removed == nil {
        removed = []RemovedItem{}
    }

    payload := struct {
        Changed []ChangedItem `json:"changed"`
        Added   []AddedItem   `json:"added"`
        Removed []RemovedItem `json:"removed"`
    }{
        Changed: changed,
        Added:   added,
        Removed: removed,
    }

    data, err := json.MarshalIndent(payload, "", "  ")
    if err != nil {
        return "", err
    }
    return string(data) + "\n", nil
}

func formatText(result DiffResult) (string, error) {
    var sb strings.Builder

    writeChanged(&sb, result.Changed)
    writeAdded(&sb, result.Added)
    writeRemoved(&sb, result.Removed)
    writeReordered(&sb, result.Reordered)

    if sb.Len() == 0 {
        return "No meaningful differences.\n", nil
    }
    return strings.TrimRight(sb.String(), "\n") + "\n", nil
}

func writeChanged(sb *strings.Builder, items []ChangedItem) {
    if len(items) == 0 {
        return
    }
    fmt.Fprintf(sb, "Changed (%d):\n", len(items))
    for _, item := range items {
        fmt.Fprintf(sb, "  %s: %s -> %s\n", item.Path, prettyJSON(item.Before), prettyJSON(item.After))
    }
    sb.WriteString("\n")
}

func writeAdded(sb *strings.Builder, items []AddedItem) {
    if len(items) == 0 {
        return
    }
    fmt.Fprintf(sb, "Added (%d):\n", len(items))
    for _, item := range items {
        fmt.Fprintf(sb, "  %s: %s\n", item.Path, prettyJSON(item.Value))
    }
    sb.WriteString("\n")
}

func writeRemoved(sb *strings.Builder, items []RemovedItem) {
    if len(items) == 0 {
        return
    }
    fmt.Fprintf(sb, "Removed (%d):\n", len(items))
    for _, item := range items {
        fmt.Fprintf(sb, "  %s\n", item.Path)
    }
    sb.WriteString("\n")
}

func writeReordered(sb *strings.Builder, items []string) {
    if len(items) == 0 {
        return
    }
    fmt.Fprintf(sb, "Unchanged (order only) (%d):\n", len(items))
    for _, item := range items {
        fmt.Fprintf(sb, "  %s\n", item)
    }
    sb.WriteString("\n")
}

func prettyJSON(v any) string {
    b, err := json.Marshal(v)
    if err != nil {
        return fmt.Sprintf("%v", v)
    }
    return string(b)
}
