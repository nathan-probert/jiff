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

var version = "dev"

type outputMode string

const (
    modeSummary outputMode = "summary"
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

    output, err := formatResult(result, opts.Mode, left, right)
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
    summary := fs.Bool("summary", false, "Summary output")
    raw := fs.Bool("raw", false, "Raw JSON diff output")
    full := fs.Bool("full", false, "Classic colorized full diff")
    versionFlag := fs.Bool("version", false, "Print version and exit")

    if err := fs.Parse(normalizeArgOrder(args)); err != nil {
        return cliOptions{}, usageError(err.Error())
    }

    if *versionFlag {
        fmt.Println(version)
        os.Exit(0)
    }

    mode, err := pickMode(*summary, *raw, *full)
    if err != nil {
        return cliOptions{}, usageError(err.Error())
    }

    positional := fs.Args()
    if len(positional) != 2 {
        return cliOptions{}, usageError("usage: jiff <file1> <file2> [--ignore fields] [--match key] [--unordered] [--summary|--raw|--full]")
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

func pickMode(summary, raw, full bool) (outputMode, error) {
    trueCount := 0
    for _, b := range []bool{summary, raw, full} {
        if b {
            trueCount++
        }
    }
    if trueCount > 1 {
        return "", errors.New("choose exactly one output mode: --summary, --raw, or --full")
    }

    if summary {
        return modeSummary, nil
    }
    if raw {
        return modeRaw, nil
    }

    // default to full mode
    return modeFull, nil
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
