package bencode

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func Decode(reader io.Reader) (any, error) {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	decoded, _, err := decodeValue(string(bytes))
	return decoded, err
}

func decodeValue(encoded string) (any, string, error) {
	if len(encoded) == 0 {
		return nil, "", nil
	}

	switch encoded[0] {
	case 'i':
		end := strings.IndexByte(encoded, 'e')
		if end == -1 {
			return nil, "", fmt.Errorf("weird integer: missing 'e'")
		}

		val, err := strconv.ParseInt(encoded[1:end], 10, 64)
		if err != nil {
			return nil, "", err
		}

		return val, encoded[end+1:], nil

	case 'l':
		var list []any
		rest := encoded[1:]

		for len(rest) > 0 && rest[0] != 'e' {
			val, remaining, err := decodeValue(rest)
			if err != nil {
				return nil, "", err
			}

			list = append(list, val)
			rest = remaining
		}

		if len(rest) == 0 {
			return nil, "", fmt.Errorf("weird list: missing 'e'")
		}

		return list, rest[1:], nil

	case 'd':
		dict := make(map[string]any)
		rest := encoded[1:]

		for len(rest) > 0 && rest[0] != 'e' {
			rawKey, remaining, err := decodeValue(rest)
			if err != nil {
				return nil, "", err
			}

			key, ok := rawKey.(string)
			if !ok {
				return nil, "", fmt.Errorf("dict keys must be strings, got %T", rawKey)
			}

			val, remaining, err := decodeValue(remaining)
			if err != nil {
				return nil, "", err
			}

			dict[key] = val
			rest = remaining
		}

		if len(rest) == 0 {
			return nil, "", fmt.Errorf("weird dict: missing 'e'")
		}

		return dict, rest[1:], nil

	default:
		if encoded[0] >= '0' && encoded[0] <= '9' {
			colon := strings.IndexByte(encoded, ':')
			if colon == -1 {
				return nil, "", fmt.Errorf("weird string: missing ':'")
			}

			length, err := strconv.Atoi(encoded[:colon])
			if err != nil {
				return nil, "", err
			}

			start := colon + 1
			end := start + length
			if end > len(encoded) {
				return nil, "", fmt.Errorf("string length out of bounds")
			}

			return encoded[start:end], encoded[end:], nil
		}

	}

	return nil, "", fmt.Errorf("weird encoded value starting with: %c", encoded[0])
}
