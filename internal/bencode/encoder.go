package bencode

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
)

func Encode(data any) ([]byte, error) {
	var buf bytes.Buffer

	err := encodeValue(&buf, data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func encodeValue(buf *bytes.Buffer, data any) error {
	switch typedVal := data.(type) {
	case string:
		fmt.Fprintf(buf, "%d:%s", len(typedVal), typedVal)

	case int, int64:
		fmt.Fprintf(buf, "i%de", typedVal)

	case []any:
		buf.WriteByte('l')
		for _, elem := range typedVal {
			if err := encodeValue(buf, elem); err != nil {
				return err
			}
		}
		buf.WriteByte('e')

	case map[string]any:
		keys := slices.Collect(maps.Keys(typedVal))
		slices.Sort(keys)

		buf.WriteByte('d')
		for _, key := range keys {
			if err := encodeValue(buf, key); err != nil {
				return err
			}
			if err := encodeValue(buf, typedVal[key]); err != nil {
				return err
			}
		}
		buf.WriteByte('e')

	default:
		return fmt.Errorf("weird type for bencoding: %T", typedVal)
	}

	return nil
}
