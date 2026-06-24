package main

import (
    "encoding/json"
    "fmt"
    "os"
    "strconv"
    "strings"
    "unicode"
)

func formatPrettyInput(input string) (string, error) {
    info, err := os.Stat(input)
    if err != nil {
        return "", fmt.Errorf("%s is not a readable file", input)
    }
    if info.IsDir() {
        return "", fmt.Errorf("%s is a directory, expected a file", input)
    }

    data, err := os.ReadFile(input)
    if err != nil {
        return "", err
    }

    value, err := parsePythonLiteral(string(data))
    if err != nil {
        return "", err
    }

    formatted, err := json.MarshalIndent(value, "", "  ")
    if err != nil {
        return "", err
    }
    return string(formatted) + "\n", nil
}

func parsePythonLiteral(input string) (any, error) {
    p := &pythonLiteralParser{input: []rune(input)}
    value, err := p.parseValue()
    if err != nil {
        return nil, err
    }
    p.skipSpace()
    if !p.eof() {
        return nil, fmt.Errorf("unexpected trailing data at position %d", p.pos+1)
    }
    return value, nil
}

type pythonLiteralParser struct {
    input []rune
    pos   int
}

func (p *pythonLiteralParser) parseValue() (any, error) {
    p.skipSpace()
    if p.eof() {
        return nil, fmt.Errorf("unexpected end of input")
    }

    switch p.peek() {
    case '{':
        return p.parseObject()
    case '[':
        return p.parseArray()
    case '"', '\'':
        return p.parseString()
    default:
        return p.parseAtom()
    }
}

func (p *pythonLiteralParser) parseObject() (any, error) {
    if err := p.expect('{'); err != nil {
        return nil, err
    }

    out := make(map[string]any)
    p.skipSpace()
    if p.accept('}') {
        return out, nil
    }

    for {
        key, err := p.parseObjectKey()
        if err != nil {
            return nil, err
        }
        p.skipSpace()
        if err := p.expect(':'); err != nil {
            return nil, err
        }
        value, err := p.parseValue()
        if err != nil {
            return nil, err
        }
        out[key] = value

        p.skipSpace()
        if p.accept('}') {
            return out, nil
        }
        if err := p.expect(','); err != nil {
            return nil, err
        }
        p.skipSpace()
        if p.accept('}') {
            return out, nil
        }
    }
}

func (p *pythonLiteralParser) parseArray() (any, error) {
    if err := p.expect('['); err != nil {
        return nil, err
    }

    out := make([]any, 0)
    p.skipSpace()
    if p.accept(']') {
        return out, nil
    }

    for {
        value, err := p.parseValue()
        if err != nil {
            return nil, err
        }
        out = append(out, value)

        p.skipSpace()
        if p.accept(']') {
            return out, nil
        }
        if err := p.expect(','); err != nil {
            return nil, err
        }
        p.skipSpace()
        if p.accept(']') {
            return out, nil
        }
    }
}

func (p *pythonLiteralParser) parseObjectKey() (string, error) {
    p.skipSpace()
    if p.eof() {
        return "", fmt.Errorf("unexpected end of input while reading object key")
    }

    if p.peek() == '"' || p.peek() == '\'' {
        return p.parseString()
    }

    start := p.pos
    for !p.eof() {
        r := p.peek()
        if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
            p.pos++
            continue
        }
        break
    }
    if start == p.pos {
        return "", fmt.Errorf("expected object key at position %d", p.pos+1)
    }
    return string(p.input[start:p.pos]), nil
}

func (p *pythonLiteralParser) parseString() (string, error) {
    quote := p.peek()
    p.pos++

    var sb strings.Builder
    for !p.eof() {
        r := p.peek()
        p.pos++
        if r == quote {
            return sb.String(), nil
        }
        if r != '\\' {
            sb.WriteRune(r)
            continue
        }
        if p.eof() {
            return "", fmt.Errorf("unterminated escape sequence")
        }
        esc := p.peek()
        p.pos++
        switch esc {
        case '\\', '\'', '"':
            sb.WriteRune(esc)
        case 'n':
            sb.WriteRune('\n')
        case 'r':
            sb.WriteRune('\r')
        case 't':
            sb.WriteRune('\t')
        case 'b':
            sb.WriteRune('\b')
        case 'f':
            sb.WriteRune('\f')
        case 'a':
            sb.WriteRune('\a')
        case 'v':
            sb.WriteRune('\v')
        case 'x':
            r, err := p.readHexRune(2)
            if err != nil {
                return "", err
            }
            sb.WriteRune(r)
        case 'u':
            r, err := p.readHexRune(4)
            if err != nil {
                return "", err
            }
            sb.WriteRune(r)
        case 'U':
            r, err := p.readHexRune(8)
            if err != nil {
                return "", err
            }
            sb.WriteRune(r)
        default:
            sb.WriteRune(esc)
        }
    }

    return "", fmt.Errorf("unterminated string literal")
}

func (p *pythonLiteralParser) parseAtom() (any, error) {
    start := p.pos
    for !p.eof() {
        r := p.peek()
        if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '+' || r == '-' || r == 'e' || r == 'E' {
            p.pos++
            continue
        }
        break
    }
    if start == p.pos {
        return nil, fmt.Errorf("unexpected token at position %d", p.pos+1)
    }

    token := string(p.input[start:p.pos])
    switch token {
    case "None", "null":
        return nil, nil
    case "True", "true":
        return true, nil
    case "False", "false":
        return false, nil
    }

    if _, err := strconv.ParseFloat(token, 64); err == nil {
        return jsonNumber(token), nil
    }

    return nil, fmt.Errorf("unexpected literal %q at position %d", token, start+1)
}

type jsonNumber string

func (n jsonNumber) MarshalJSON() ([]byte, error) {
    return []byte(n), nil
}

func (p *pythonLiteralParser) readHexRune(width int) (rune, error) {
    if p.pos+width > len(p.input) {
        return 0, fmt.Errorf("incomplete escape sequence at position %d", p.pos+1)
    }
    value := 0
    for i := 0; i < width; i++ {
        digit := p.input[p.pos+i]
        v, ok := fromHexDigit(digit)
        if !ok {
            return 0, fmt.Errorf("invalid escape sequence at position %d", p.pos+1)
        }
        value = value*16 + v
    }
    p.pos += width
    return rune(value), nil
}

func fromHexDigit(r rune) (int, bool) {
    switch {
    case '0' <= r && r <= '9':
        return int(r - '0'), true
    case 'a' <= r && r <= 'f':
        return int(r-'a') + 10, true
    case 'A' <= r && r <= 'F':
        return int(r-'A') + 10, true
    default:
        return 0, false
    }
}

func (p *pythonLiteralParser) skipSpace() {
    for !p.eof() && unicode.IsSpace(p.peek()) {
        p.pos++
    }
}

func (p *pythonLiteralParser) accept(expected rune) bool {
    if p.eof() || p.peek() != expected {
        return false
    }
    p.pos++
    return true
}

func (p *pythonLiteralParser) expect(expected rune) error {
    if p.accept(expected) {
        return nil
    }
    if p.eof() {
        return fmt.Errorf("expected %q but reached end of input", expected)
    }
    return fmt.Errorf("expected %q at position %d", expected, p.pos+1)
}

func (p *pythonLiteralParser) peek() rune {
    return p.input[p.pos]
}

func (p *pythonLiteralParser) eof() bool {
    return p.pos >= len(p.input)
}