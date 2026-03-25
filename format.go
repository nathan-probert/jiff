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
    case modeVerbose:
        return formatText(result, true), nil
    case modeSummary:
        return formatText(result, false), nil
    default:
        return "", fmt.Errorf("unknown output mode: %s", mode)
    }
}

func formatRaw(result DiffResult) (string, error) {
    b, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return "", err
    }
    return string(b) + "\n", nil
}

func formatText(result DiffResult, verbose bool) string {
    var sb strings.Builder

    writeChanged(&sb, result.Changed)
    writeAdded(&sb, result.Added)
    writeRemoved(&sb, result.Removed, verbose)
    writeReordered(&sb, result.Reordered)

    if sb.Len() == 0 {
        return "No meaningful differences.\n"
    }
    return strings.TrimRight(sb.String(), "\n") + "\n"
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

func writeRemoved(sb *strings.Builder, items []RemovedItem, verbose bool) {
    if len(items) == 0 {
        return
    }
    fmt.Fprintf(sb, "Removed (%d):\n", len(items))
    for _, item := range items {
        if verbose {
            fmt.Fprintf(sb, "  %s: %s\n", item.Path, prettyJSON(item.Value))
        } else {
            fmt.Fprintf(sb, "  %s\n", item.Path)
        }
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
