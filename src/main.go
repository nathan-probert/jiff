package main

import (
    "encoding/json"
    "errors"
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
        errorAndExit(err.Error())
    }

    left, err := parseJSONFile(opts.FileA)
    if err != nil {
        errorAndExit(fmt.Sprintf("failed to parse %s: %v", opts.FileA, err))
    }

    right, err := parseJSONFile(opts.FileB)
    if err != nil {
        errorAndExit(fmt.Sprintf("failed to parse %s: %v", opts.FileB, err))
    }

    left = removeIgnoredFields(left, toSet(opts.IgnoreFields))
    right = removeIgnoredFields(right, toSet(opts.IgnoreFields))

    result := diffValues(left, right, DiffOptions{
        MatchKey:  opts.MatchKey,
        Unordered: opts.Unordered,
    })

    output, err := formatResult(result, opts.Mode, left, right)
    if err != nil {
        errorAndExit(fmt.Sprintf("failed to format diff: %v", err))
    }
    fmt.Print(output)
}
