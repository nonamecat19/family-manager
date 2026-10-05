package tgemu

import (
	"fmt"
	"strings"
	"unicode/utf16"
)

const maxTextLength = 4096

var allowedTags = map[string]bool{
	"b": true, "strong": true,
	"i": true, "em": true,
	"u": true, "ins": true,
	"s": true, "strike": true, "del": true,
	"span": true, "tg-spoiler": true,
	"a": true, "code": true, "pre": true,
	"blockquote": true, "tg-emoji": true,
}

var namedEntities = map[string]string{"lt": "<", "gt": ">", "amp": "&", "quot": "\""}

func parseHTML(s string) (string, error) {
	var plain strings.Builder
	var stack []string
	for i := 0; i < len(s); {
		switch s[i] {
		case '<':
			end := strings.IndexByte(s[i:], '>')
			if end < 0 {
				return "", fmt.Errorf("can't parse entities: Unclosed start tag at byte offset %d", i)
			}
			raw := s[i+1 : i+end]
			if closing, ok := strings.CutPrefix(raw, "/"); ok {
				name := strings.ToLower(strings.TrimSpace(closing))
				if len(stack) == 0 {
					return "", fmt.Errorf("can't parse entities: Unexpected end tag at byte offset %d", i)
				}
				top := stack[len(stack)-1]
				if name != top {
					return "", fmt.Errorf(
						"can't parse entities: Unmatched end tag at byte offset %d, expected \"</%s>\", found \"</%s>\"",
						i, top, name)
				}
				stack = stack[:len(stack)-1]
			} else {
				name := strings.ToLower(raw)
				if sp := strings.IndexAny(name, " \t\n"); sp >= 0 {
					name = name[:sp]
				}
				if !allowedTags[name] {
					return "", fmt.Errorf(
						"can't parse entities: Unsupported start tag \"%s\" at byte offset %d", name, i)
				}
				if name == "a" && !strings.Contains(strings.ToLower(raw), "href=") {
					return "", fmt.Errorf("can't parse entities: Can't find href in a tag at byte offset %d", i)
				}
				stack = append(stack, name)
			}
			i += end + 1
		case '>':
			plain.WriteByte('>')
			i++
		case '&':
			semi := strings.IndexByte(s[i:], ';')
			if semi > 1 && semi <= 10 {
				name := s[i+1 : i+semi]
				if v, ok := namedEntities[name]; ok {
					plain.WriteString(v)
					i += semi + 1
					continue
				}
				if strings.HasPrefix(name, "#") {
					var r rune
					var err error
					if strings.HasPrefix(name, "#x") || strings.HasPrefix(name, "#X") {
						_, err = fmt.Sscanf(name[2:], "%x", &r)
					} else {
						_, err = fmt.Sscanf(name[1:], "%d", &r)
					}
					if err == nil {
						plain.WriteRune(r)
						i += semi + 1
						continue
					}
				}
			}
			plain.WriteByte('&')
			i++
		default:
			plain.WriteByte(s[i])
			i++
		}
	}
	if len(stack) > 0 {
		return "", fmt.Errorf(
			"can't parse entities: Can't find end tag corresponding to start tag \"%s\"", stack[len(stack)-1])
	}
	return plain.String(), nil
}

func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func checkText(text, parseMode string) error {
	plain := text
	if strings.EqualFold(parseMode, "HTML") {
		p, err := parseHTML(text)
		if err != nil {
			return err
		}
		plain = p
	}
	if strings.TrimSpace(plain) == "" {
		return fmt.Errorf("message text is empty")
	}
	if utf16Len(plain) > maxTextLength {
		return fmt.Errorf("message is too long")
	}
	return nil
}

func checkMarkup(m *Markup) error {
	if m == nil {
		return nil
	}
	for _, row := range m.InlineKeyboard {
		for _, b := range row {
			if b.Text == "" {
				return fmt.Errorf("BUTTON_TEXT_EMPTY")
			}
			hasData, hasURL := b.CallbackData != "", b.URL != ""
			if hasData == hasURL {
				return fmt.Errorf("can't parse inline keyboard button: Text buttons are unallowed in the inline keyboard")
			}
			if hasData && len(b.CallbackData) > 64 {
				return fmt.Errorf("BUTTON_DATA_INVALID")
			}
			if hasURL && !(strings.HasPrefix(b.URL, "https://") || strings.HasPrefix(b.URL, "http://") ||
				strings.HasPrefix(b.URL, "tg://")) {
				return fmt.Errorf("inline keyboard button URL '%s' is invalid: Unsupported URL protocol", b.URL)
			}
		}
	}
	return nil
}
