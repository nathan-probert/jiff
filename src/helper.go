package main;

import (
	"strings" 
	"flag" 
	"io" 
	"fmt" 
	"os"
    "encoding/json"
    "errors"
)

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

func parseFlags(args []string) (cliOptions, error) {
    fs := flag.NewFlagSet("jiff", flag.ContinueOnError)
    fs.SetOutput(io.Discard)

    prettyInput := fs.String("p", "", "Pretty print a file and exit")
    fs.StringVar(prettyInput, "pretty", "", "Pretty print a file and exit")

    ignoreCSV := fs.String("i", "", "Comma-separated list of fields to ignore recursively")
    fs.StringVar(ignoreCSV, "ignore", "", "Comma-separated list of fields to ignore recursively")

    matchKey := fs.String("m", "", "Key used to match objects inside arrays")
    fs.StringVar(matchKey, "match", "", "Key used to match objects inside arrays")

    unordered := fs.Bool("u", false, "Treat arrays as unordered")
    fs.BoolVar(unordered, "unordered", false, "Treat arrays as unordered")

    summary := fs.Bool("s", false, "Summary output")
    fs.BoolVar(summary, "summary", false, "Summary output")

    raw := fs.Bool("r", false, "Raw JSON diff output")
    fs.BoolVar(raw, "raw", false, "Raw JSON diff output")

    full := fs.Bool("f", false, "Classic colorized full diff")
    fs.BoolVar(full, "full", false, "Classic colorized full diff")

    versionFlag := fs.Bool("v", false, "Print version and exit")
    fs.BoolVar(versionFlag, "version", false, "Print version and exit")

    err := fs.Parse(normalizeArgOrder(args))
    if err != nil {
        return cliOptions{}, usageError(err.Error())
    }

    if *versionFlag {
        fmt.Println(version)
        os.Exit(0)
    }

    pretty := strings.TrimSpace(*prettyInput)
    if pretty != "" {
        if len(fs.Args()) != 0 {
            return cliOptions{}, usageError("usage: jiff -p <file>")
        }
        return cliOptions{PrettyInput: pretty}, nil
    }

    mode, err := pickMode(*summary, *raw, *full)
    if err != nil {
        return cliOptions{}, usageError(err.Error())
    }

    positional := fs.Args()
    if len(positional) != 2 {
        return cliOptions{}, usageError("usage: jiff [-p file] <file1> <file2> [--ignore fields] [--match key] [--unordered] [--summary|--raw|--full]")
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

func flagNeedsValue(name string) bool {
    switch name {
    case "-p", "--pretty":
        return true
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

func errorAndExit(msg string) {
    fmt.Fprintln(os.Stderr, msg)
    os.Exit(1)
}
