package main

import (
    "encoding/json"
    "fmt"
    "reflect"
    "sort"
)

type DiffOptions struct {
    MatchKey  string
    Unordered bool
}

type ChangedItem struct {
    Path   string `json:"path"`
    Before any    `json:"before"`
    After  any    `json:"after"`
}

type AddedItem struct {
    Path  string `json:"path"`
    Value any    `json:"value"`
}

type RemovedItem struct {
    Path  string `json:"path"`
    Value any    `json:"value"`
}

type DiffResult struct {
    Changed   []ChangedItem `json:"changed"`
    Added     []AddedItem   `json:"added"`
    Removed   []RemovedItem `json:"removed"`
    Reordered []string      `json:"reordered"`
}

func diffValues(a, b any, opts DiffOptions) DiffResult {
    var result DiffResult
    walkDiff("", a, b, opts, &result)
    sortResult(&result)
    return result
}

func walkDiff(path string, a, b any, opts DiffOptions, out *DiffResult) {
    if reflect.DeepEqual(a, b) {
        return
    }

    if a == nil && b != nil {
        out.Added = append(out.Added, AddedItem{Path: pathOrRoot(path), Value: b})
        return
    }
    if b == nil && a != nil {
        out.Removed = append(out.Removed, RemovedItem{Path: pathOrRoot(path), Value: a})
        return
    }

    aMap, aIsMap := a.(map[string]any)
    bMap, bIsMap := b.(map[string]any)
    if aIsMap || bIsMap {
        if !aIsMap || !bIsMap {
            out.Changed = append(out.Changed, ChangedItem{Path: pathOrRoot(path), Before: a, After: b})
            return
        }
        diffMaps(path, aMap, bMap, opts, out)
        return
    }

    aArr, aIsArr := a.([]any)
    bArr, bIsArr := b.([]any)
    if aIsArr || bIsArr {
        if !aIsArr || !bIsArr {
            out.Changed = append(out.Changed, ChangedItem{Path: pathOrRoot(path), Before: a, After: b})
            return
        }
        diffArrays(path, aArr, bArr, opts, out)
        return
    }

    out.Changed = append(out.Changed, ChangedItem{Path: pathOrRoot(path), Before: a, After: b})
}

func diffMaps(path string, a, b map[string]any, opts DiffOptions, out *DiffResult) {
    keys := make(map[string]struct{}, len(a)+len(b))
    for k := range a {
        keys[k] = struct{}{}
    }
    for k := range b {
        keys[k] = struct{}{}
    }

    ordered := make([]string, 0, len(keys))
    for k := range keys {
        ordered = append(ordered, k)
    }
    sort.Strings(ordered)

    for _, k := range ordered {
        p := joinPath(path, k)
        av, aok := a[k]
        bv, bok := b[k]

        switch {
        case !aok && bok:
            out.Added = append(out.Added, AddedItem{Path: p, Value: bv})
        case aok && !bok:
            out.Removed = append(out.Removed, RemovedItem{Path: p, Value: av})
        default:
            walkDiff(p, av, bv, opts, out)
        }
    }
}

func diffArrays(path string, a, b []any, opts DiffOptions, out *DiffResult) {
    if opts.MatchKey != "" && allObjectsWithKey(a, opts.MatchKey) && allObjectsWithKey(b, opts.MatchKey) {
        diffArrayByMatch(path, a, b, opts, out)
        return
    }

    if opts.Unordered {
        diffArrayUnordered(path, a, b, out)
        return
    }

    if sameMultiset(a, b) && !sameSequence(a, b) {
        out.Reordered = append(out.Reordered, pathOrRoot(path))
        return
    }

    diffArrayIndexed(path, a, b, opts, out)
}

func diffArrayIndexed(path string, a, b []any, opts DiffOptions, out *DiffResult) {
    n := len(a)
    if len(b) < n {
        n = len(b)
    }
    for i := 0; i < n; i++ {
        walkDiff(joinIndex(path, i), a[i], b[i], opts, out)
    }
    for i := n; i < len(a); i++ {
        out.Removed = append(out.Removed, RemovedItem{Path: joinIndex(path, i), Value: a[i]})
    }
    for i := n; i < len(b); i++ {
        out.Added = append(out.Added, AddedItem{Path: joinIndex(path, i), Value: b[i]})
    }
}

func diffArrayByMatch(path string, a, b []any, opts DiffOptions, out *DiffResult) {
    mapA, orderA, uniqueA := indexByMatchKey(a, opts.MatchKey)
    mapB, orderB, uniqueB := indexByMatchKey(b, opts.MatchKey)
    if !uniqueA || !uniqueB {
        // Duplicate match keys make identity ambiguous, so fall back to index-based comparison.
        fallback := opts
        fallback.MatchKey = ""
        diffArrayIndexed(path, a, b, fallback, out)
        return
    }

    keys := make(map[string]struct{}, len(mapA)+len(mapB))
    for k := range mapA {
        keys[k] = struct{}{}
    }
    for k := range mapB {
        keys[k] = struct{}{}
    }

    ordered := make([]string, 0, len(keys))
    for k := range keys {
        ordered = append(ordered, k)
    }
    sort.Strings(ordered)

    for _, k := range ordered {
        av, aok := mapA[k]
        bv, bok := mapB[k]
        elemPath := joinMatch(path, opts.MatchKey, k)

        switch {
        case !aok && bok:
            out.Added = append(out.Added, AddedItem{Path: elemPath, Value: bv})
        case aok && !bok:
            out.Removed = append(out.Removed, RemovedItem{Path: elemPath, Value: av})
        default:
            walkDiff(elemPath, av, bv, opts, out)
        }
    }

    if sameStringSlice(orderA, orderB) {
        return
    }
    if sameSet(orderA, orderB) {
        out.Reordered = append(out.Reordered, pathOrRoot(path))
    }
}

func diffArrayUnordered(path string, a, b []any, out *DiffResult) {
    addedBefore := len(out.Added)
    removedBefore := len(out.Removed)

    countsA := multisetCounts(a)
    countsB := multisetCounts(b)

    keys := make(map[string]struct{}, len(countsA)+len(countsB))
    for k := range countsA {
        keys[k] = struct{}{}
    }
    for k := range countsB {
        keys[k] = struct{}{}
    }

    ordered := make([]string, 0, len(keys))
    for k := range keys {
        ordered = append(ordered, k)
    }
    sort.Strings(ordered)

    for _, key := range ordered {
        aCount := countsA[key]
        bCount := countsB[key]
        switch {
        case aCount > bCount:
            for i := 0; i < aCount-bCount; i++ {
                out.Removed = append(out.Removed, RemovedItem{Path: pathOrRoot(path), Value: canonicalDecode(key)})
            }
        case bCount > aCount:
            for i := 0; i < bCount-aCount; i++ {
                out.Added = append(out.Added, AddedItem{Path: pathOrRoot(path), Value: canonicalDecode(key)})
            }
        }
    }

    if len(out.Added) == addedBefore && len(out.Removed) == removedBefore && !sameSequence(a, b) {
        out.Reordered = append(out.Reordered, pathOrRoot(path))
    }
}

func removeIgnoredFields(v any, ignore map[string]struct{}) any {
    if len(ignore) == 0 {
        return v
    }

    switch t := v.(type) {
    case map[string]any:
        cleaned := make(map[string]any, len(t))
        for k, child := range t {
            if _, skipped := ignore[k]; skipped {
                continue
            }
            cleaned[k] = removeIgnoredFields(child, ignore)
        }
        return cleaned
    case []any:
        cleaned := make([]any, len(t))
        for i, child := range t {
            cleaned[i] = removeIgnoredFields(child, ignore)
        }
        return cleaned
    default:
        return v
    }
}

func allObjectsWithKey(arr []any, key string) bool {
    if len(arr) == 0 {
        return true
    }
    for _, item := range arr {
        m, ok := item.(map[string]any)
        if !ok {
            return false
        }
        if _, exists := m[key]; !exists {
            return false
        }
    }
    return true
}

func indexByMatchKey(arr []any, key string) (map[string]any, []string, bool) {
    out := make(map[string]any, len(arr))
    order := make([]string, 0, len(arr))
    for _, item := range arr {
        m := item.(map[string]any)
        raw := m[key]
        id := canonicalKey(raw)
        if _, exists := out[id]; exists {
            return nil, nil, false
        }
        out[id] = item
        order = append(order, id)
    }
    return out, order, true
}

func canonicalKey(v any) string {
    if b, err := json.Marshal(v); err == nil {
        return string(b)
    }
    return fmt.Sprintf("%v", v)
}

func canonicalString(v any) string {
    b, err := json.Marshal(v)
    if err != nil {
        return fmt.Sprintf("%T:%v", v, v)
    }
    return string(b)
}

func canonicalDecode(s string) any {
    var v any
    if err := json.Unmarshal([]byte(s), &v); err != nil {
        return s
    }
    return v
}

func multisetCounts(arr []any) map[string]int {
    out := make(map[string]int, len(arr))
    for _, item := range arr {
        out[canonicalString(item)]++
    }
    return out
}

func sameMultiset(a, b []any) bool {
    if len(a) != len(b) {
        return false
    }
    return reflect.DeepEqual(multisetCounts(a), multisetCounts(b))
}

func sameSequence(a, b []any) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if canonicalString(a[i]) != canonicalString(b[i]) {
            return false
        }
    }
    return true
}

func sameStringSlice(a, b []string) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }
    return true
}

func sameSet(a, b []string) bool {
    if len(a) != len(b) {
        return false
    }
    m := make(map[string]int, len(a))
    for _, item := range a {
        m[item]++
    }
    for _, item := range b {
        m[item]--
    }
    for _, v := range m {
        if v != 0 {
            return false
        }
    }
    return true
}

func joinPath(base, child string) string {
    if base == "" {
        return child
    }
    return base + "." + child
}

func joinIndex(base string, idx int) string {
    if base == "" {
        return fmt.Sprintf("root[%d]", idx)
    }
    return fmt.Sprintf("%s[%d]", base, idx)
}

func joinMatch(base, key, id string) string {
    if base == "" {
        return fmt.Sprintf("root[%s=%s]", key, id)
    }
    return fmt.Sprintf("%s[%s=%s]", base, key, id)
}

func pathOrRoot(path string) string {
    if path == "" {
        return "root"
    }
    return path
}

func sortResult(res *DiffResult) {
    sort.Slice(res.Changed, func(i, j int) bool {
        return res.Changed[i].Path < res.Changed[j].Path
    })
    sort.Slice(res.Added, func(i, j int) bool {
        return res.Added[i].Path < res.Added[j].Path
    })
    sort.Slice(res.Removed, func(i, j int) bool {
        return res.Removed[i].Path < res.Removed[j].Path
    })
    sort.Strings(res.Reordered)
}
