package main

import (
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "os"
    "strings"
)

type outputMode string

const (
    modeSummary outputMode = "summary"
    modeVerbose outputMode = "verbose"
    modeRaw     outputMode = "raw"
    modeFull    outputMode = "full"
)

type cliOptions struct {
    IgnoreFields []string
    MatchKey     string
    Unordered    bool
    Mode         outputMode
    FileA        string
    FileB        string
}

func main() {
    opts, err := parseFlags(os.Args[1:])
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(2)
    }

    left, err := parseJSONFile(opts.FileA)
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to parse %s: %v\n", opts.FileA, err)
        os.Exit(1)
    }

    right, err := parseJSONFile(opts.FileB)
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to parse %s: %v\n", opts.FileB, err)
        os.Exit(1)
    }

    left = removeIgnoredFields(left, toSet(opts.IgnoreFields))
    right = removeIgnoredFields(right, toSet(opts.IgnoreFields))

    result := diffValues(left, right, DiffOptions{
        MatchKey:  opts.MatchKey,
        Unordered: opts.Unordered,
    })

    if opts.Mode == modeFull {
        output, err := formatFullDiff(left, right)
        if err != nil {
            fmt.Fprintf(os.Stderr, "failed to format full diff: %v\n", err)
            os.Exit(1)
        }
        fmt.Print(output)
        return
    }

    output, err := formatResult(result, opts.Mode)
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to format diff: %v\n", err)
        os.Exit(1)
    }

    fmt.Print(output)
}

func parseFlags(args []string) (cliOptions, error) {
    fs := flag.NewFlagSet("jiff", flag.ContinueOnError)
    fs.SetOutput(io.Discard)

    ignoreCSV := fs.String("ignore", "", "Comma-separated list of fields to ignore recursively")
    matchKey := fs.String("match", "", "Key used to match objects inside arrays")
    unordered := fs.Bool("unordered", false, "Treat arrays as unordered")
    summary := fs.Bool("summary", false, "Summary (default) output")
    verbose := fs.Bool("verbose", false, "Verbose output with full values")
    raw := fs.Bool("raw", false, "Raw JSON diff output")
    full := fs.Bool("full", false, "Classic colorized full diff")

    if err := fs.Parse(normalizeArgOrder(args)); err != nil {
        return cliOptions{}, usageError(err.Error())
    }

    mode, err := pickMode(*summary, *verbose, *raw, *full)
    if err != nil {
        return cliOptions{}, usageError(err.Error())
    }

    positional := fs.Args()
    if len(positional) != 2 {
        return cliOptions{}, usageError("usage: jiff <file1> <file2> [--ignore fields] [--match key] [--unordered] [--summary|--verbose|--raw|--full]")
    }

    return cliOptions{
        IgnoreFields: parseIgnoreCSV(*ignoreCSV),
        MatchKey:     strings.TrimSpace(*matchKey),
        Unordered:    *unordered,
        Mode:         mode,
        FileA:        positional[0],
        FileB:        positional[1],
    }, nil
}

func normalizeArgOrder(args []string) []string {
    if len(args) == 0 {
        return args
    }

    flags := make([]string, 0, len(args))
    positional := make([]string, 0, len(args))

    for i := 0; i < len(args); i++ {
        arg := args[i]

        if arg == "--" {
            positional = append(positional, args[i+1:]...)
            break
        }

        if !strings.HasPrefix(arg, "-") || arg == "-" {
            positional = append(positional, arg)
            continue
        }

        flags = append(flags, arg)

        name := arg
        if eq := strings.Index(arg, "="); eq >= 0 {
            name = arg[:eq]
        }

        if strings.HasPrefix(name, "--") && flagNeedsValue(name) && !strings.Contains(arg, "=") && i+1 < len(args) {
            i++
            flags = append(flags, args[i])
        }
    }

    out := make([]string, 0, len(args))
    out = append(out, flags...)
    out = append(out, positional...)
    return out
}

func flagNeedsValue(name string) bool {
    switch name {
    case "--ignore", "--match":
        return true
    default:
        return false
    }
}

func usageError(msg string) error {
    return errors.New(msg)
}

func pickMode(summary, verbose, raw, full bool) (outputMode, error) {
    selected := 0
    if summary {
        selected++
    }
    if verbose {
        selected++
    }
    if raw {
        selected++
    }
    if full {
        selected++
    }

    if selected > 1 {
        return "", errors.New("choose only one output mode: --summary, --verbose, --raw, or --full")
    }
    if full {
        return modeFull, nil
    }
    if raw {
        return modeRaw, nil
    }
    if verbose {
        return modeVerbose, nil
    }
    return modeSummary, nil
}

func parseIgnoreCSV(csv string) []string {
    if strings.TrimSpace(csv) == "" {
        return nil
    }

    fields := strings.Split(csv, ",")
    out := make([]string, 0, len(fields))
    for _, field := range fields {
        trimmed := strings.TrimSpace(field)
        if trimmed == "" {
            continue
        }
        out = append(out, trimmed)
    }
    return out
}

func parseJSONFile(path string) (any, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    dec := json.NewDecoder(f)
    dec.UseNumber()

    var v any
    if err := dec.Decode(&v); err != nil {
        return nil, err
    }
    var extra any
    if err := dec.Decode(&extra); err != io.EOF {
        if err == nil {
            return nil, errors.New("file contains multiple JSON values")
        }
        return nil, err
    }
    return v, nil
}

func toSet(items []string) map[string]struct{} {
    if len(items) == 0 {
        return nil
    }
    out := make(map[string]struct{}, len(items))
    for _, item := range items {
        out[item] = struct{}{}
    }
    return out
}
