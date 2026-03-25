package main

import (
    "encoding/json"
    "fmt"
    "strings"
)

const (
    ansiReset = "\x1b[0m"
    ansiDim   = "\x1b[2m"
    ansiRed   = "\x1b[31m"
    ansiGreen = "\x1b[32m"
)

type lineOp struct {
    kind string
    line string
}

func formatFullDiff(left, right any) (string, error) {
    leftLines, err := prettyLines(left)
    if err != nil {
        return "", err
    }
    rightLines, err := prettyLines(right)
    if err != nil {
        return "", err
    }

    ops := diffLines(leftLines, rightLines)

    var sb strings.Builder
    sb.WriteString(ansiDim)
    sb.WriteString("--- left\n")
    sb.WriteString("+++ right\n")
    sb.WriteString(ansiReset)

    for _, op := range ops {
        switch op.kind {
        case "equal":
            sb.WriteString("  ")
            sb.WriteString(op.line)
            sb.WriteByte('\n')
        case "delete":
            sb.WriteString(ansiRed)
            sb.WriteString("- ")
            sb.WriteString(op.line)
            sb.WriteString(ansiReset)
            sb.WriteByte('\n')
        case "add":
            sb.WriteString(ansiGreen)
            sb.WriteString("+ ")
            sb.WriteString(op.line)
            sb.WriteString(ansiReset)
            sb.WriteByte('\n')
        }
    }

    return sb.String(), nil
}

func prettyLines(v any) ([]string, error) {
    b, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return nil, fmt.Errorf("marshal pretty json: %w", err)
    }
    s := strings.ReplaceAll(string(b), "\r\n", "\n")
    if s == "" {
        return []string{""}, nil
    }
    return strings.Split(s, "\n"), nil
}

func diffLines(a, b []string) []lineOp {
    n := len(a)
    m := len(b)

    dp := make([][]int, n+1)
    for i := range dp {
        dp[i] = make([]int, m+1)
    }

    for i := n - 1; i >= 0; i-- {
        for j := m - 1; j >= 0; j-- {
            if a[i] == b[j] {
                dp[i][j] = dp[i+1][j+1] + 1
            } else if dp[i+1][j] >= dp[i][j+1] {
                dp[i][j] = dp[i+1][j]
            } else {
                dp[i][j] = dp[i][j+1]
            }
        }
    }

    i, j := 0, 0
    ops := make([]lineOp, 0, n+m)
    for i < n && j < m {
        if a[i] == b[j] {
            ops = append(ops, lineOp{kind: "equal", line: a[i]})
            i++
            j++
            continue
        }
        if dp[i+1][j] >= dp[i][j+1] {
            ops = append(ops, lineOp{kind: "delete", line: a[i]})
            i++
            continue
        }
        ops = append(ops, lineOp{kind: "add", line: b[j]})
        j++
    }

    for i < n {
        ops = append(ops, lineOp{kind: "delete", line: a[i]})
        i++
    }
    for j < m {
        ops = append(ops, lineOp{kind: "add", line: b[j]})
        j++
    }

    return ops
}
