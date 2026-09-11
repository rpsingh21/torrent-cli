package bencode

import (
	"fmt"
	"sort"
	"strconv"
)

type Encoder struct {
	buf []byte
}

// NewEncoder creates an encoder with an optional initial buffer capacity.
func NewEncoder(capacity int) *Encoder {
	return &Encoder{
		buf: make([]byte, 0, capacity),
	}
}

// Encode encodes a bencode value.
func Encode(data any) ([]byte, error) {
	encoder := NewEncoder(0)

	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	return encoder.Bytes(), nil
}

// Encode encodes data into the encoder's internal buffer.
func (e *Encoder) Encode(data any) error {
	return e.encodeValue(data)
}

// Bytes returns the encoded data.
//
// The returned slice is only valid until the encoder is reused.
func (e *Encoder) Bytes() []byte {
	return e.buf
}

func (e *Encoder) encodeValue(data any) error {
	switch value := data.(type) {
	case int:
		e.encodeInt(int64(value))

	case int8:
		e.encodeInt(int64(value))

	case int16:
		e.encodeInt(int64(value))

	case int32:
		e.encodeInt(int64(value))

	case int64:
		e.encodeInt(value)

	case []byte:
		e.encodeBytes(value)

	case []any:
		return e.encodeList(value)

	case map[string]any:
		return e.encodeDict(value)

	default:
		return fmt.Errorf("unsupported type: %T", data)
	}

	return nil
}

func (e *Encoder) encodeInt(value int64) {
	e.buf = append(e.buf, IntToken)
	e.buf = strconv.AppendInt(e.buf, value, 10)
	e.buf = append(e.buf, EndToken)
}

func (e *Encoder) encodeBytes(value []byte) {
	e.buf = strconv.AppendInt(e.buf, int64(len(value)), 10)
	e.buf = append(e.buf, SeparatorToken)
	e.buf = append(e.buf, value...)
}

func (e *Encoder) encodeList(value []any) error {
	e.buf = append(e.buf, ListToken)

	for _, element := range value {
		if err := e.encodeValue(element); err != nil {
			return fmt.Errorf("encode list element: %w", err)
		}
	}

	e.buf = append(e.buf, EndToken)

	return nil
}

func (e *Encoder) encodeDict(value map[string]any) error {
	e.buf = append(e.buf, DictToken)

	keys := make([]string, 0, len(value))

	for key := range value {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		// Dictionary keys are bencoded strings.
		e.buf = strconv.AppendInt(e.buf, int64(len(key)), 10)
		e.buf = append(e.buf, SeparatorToken)
		e.buf = append(e.buf, key...)

		if err := e.encodeValue(value[key]); err != nil {
			return fmt.Errorf("encode dictionary value for key %q: %w", key, err)
		}
	}

	e.buf = append(e.buf, EndToken)

	return nil
}
