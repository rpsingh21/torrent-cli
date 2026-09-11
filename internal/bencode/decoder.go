package bencode

import "fmt"

const (
	DictToken      = byte('d')
	ListToken      = byte('l')
	IntToken       = byte('i')
	EndToken       = byte('e')
	SeparatorToken = byte(':')
)

type Decoder struct {
	torrentData []byte
	index       int
}

func NewDecoder(data []byte) *Decoder {
	return &Decoder{
		torrentData: data,
	}
}

// Decode decodes the next bencode value.
func (d *Decoder) Decode() (any, error) {
	if d.index >= len(d.torrentData) {
		return nil, fmt.Errorf("invalid data: unexpected end at %d", d.index)
	}

	switch d.torrentData[d.index] {
	case IntToken:
		d.index++
		return d.decodeInt()

	case ListToken:
		d.index++
		return d.decodeList()

	case DictToken:
		d.index++
		return d.decodeDict()

	default:
		c := d.torrentData[d.index]

		if c >= '0' && c <= '9' {
			return d.decodeString()
		}

		return nil, fmt.Errorf("unexpected token %q at %d", c, d.index)
	}
}

// decodeInt parses:
//
//	i42e
//	i-42e
//	i0e
//
// directly from the input buffer.
func (d *Decoder) decodeInt() (int64, error) {
	if d.index >= len(d.torrentData) {
		return 0, fmt.Errorf("unterminated integer at %d", d.index)
	}

	negative := false

	if d.torrentData[d.index] == '-' {
		negative = true
		d.index++

		if d.index >= len(d.torrentData) {
			return 0, fmt.Errorf("invalid integer at %d", d.index)
		}
	}

	var value int64
	digits := 0

	for d.index < len(d.torrentData) {
		c := d.torrentData[d.index]

		if c == EndToken {
			if digits == 0 {
				return 0, fmt.Errorf("invalid integer at %d", d.index)
			}
			d.index++

			if negative {
				return -value, nil
			}
			return value, nil
		}

		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer at %d", d.index)
		}

		// Detect int64 overflow before multiplication.
		digit := int64(c - '0')

		if value > (1<<63-1-digit)/10 {
			return 0, fmt.Errorf("integer overflow at %d", d.index)
		}

		value = value*10 + digit
		d.index++
		digits++
	}

	return 0, fmt.Errorf("unterminated integer at %d", d.index)
}

// decodeString parses:
//
//	4:spam
//	0:
//	5:hello
//
// directly from the input buffer.
//
// The returned []byte references the original torrentData buffer.
func (d *Decoder) decodeString() ([]byte, error) {
	length, err := d.decodeStringLength()
	if err != nil {
		return nil, err
	}

	start := d.index

	if length > len(d.torrentData)-start {
		return nil, fmt.Errorf("string exceeds input at %d: length=%d", start, length)
	}

	d.index += length
	return d.torrentData[start:d.index], nil
}

// decodeStringLength parses the decimal length before ':'.
//
// Example:
//
//	5:hello
//	^
//	length = 5
func (d *Decoder) decodeStringLength() (int, error) {
	if d.index >= len(d.torrentData) {
		return 0, fmt.Errorf("unterminated string length at %d", d.index)
	}

	length := 0
	digits := 0

	for d.index < len(d.torrentData) {
		c := d.torrentData[d.index]

		if c == SeparatorToken {
			if digits == 0 {
				return 0, fmt.Errorf("empty string length at %d", d.index)
			}
			d.index++
			return length, nil
		}

		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid string length at %d", d.index)
		}

		digit := int(c - '0')

		// Prevent int overflow.
		if length > (int(^uint(0)>>1)-digit)/10 {
			return 0, fmt.Errorf("string length overflow at %d", d.index)
		}

		length = length*10 + digit
		d.index++
		digits++
	}

	return 0, fmt.Errorf("unterminated string length at %d", d.index)
}

func (d *Decoder) decodeList() ([]any, error) {
	values := make([]any, 0, 2)

	for d.index < len(d.torrentData) {
		if d.torrentData[d.index] == EndToken {
			d.index++
			return values, nil
		}

		value, err := d.Decode()
		if err != nil {
			return nil, fmt.Errorf("decode list element at %d: %w", d.index, err)
		}

		values = append(values, value)
	}

	return nil, fmt.Errorf("unterminated list at %d", d.index)
}

func (d *Decoder) decodeDict() (map[string]any, error) {
	values := make(map[string]any)

	for d.index < len(d.torrentData) {
		if d.torrentData[d.index] == EndToken {
			d.index++
			return values, nil
		}

		key, err := d.decodeString()
		if err != nil {
			return nil, fmt.Errorf("decode dictionary key at %d: %w", d.index, err)
		}

		value, err := d.Decode()
		if err != nil {
			return nil, fmt.Errorf("decode dictionary value for key %q at %d: %w", key, d.index, err)
		}

		values[string(key)] = value
	}

	return nil, fmt.Errorf("unterminated dictionary at %d", d.index)
}
