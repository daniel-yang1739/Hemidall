package wire

import "fmt"

const (
	varintWireType          = 0
	fixed64WireType         = 1
	lengthDelimitedWireType = 2
	fixed32WireType         = 5
	protobufFieldShift      = 3
	protobufWireMask        = 7
	fixed64ByteLength       = 8
	fixed32ByteLength       = 4
	maximumVarintShift      = 64
)

// Field is one schema-less protobuf wire field. Its number and wire type are
// observed facts; semantic names are intentionally not inferred here.
type Field struct {
	Number   int
	WireType int
	Bytes    []byte
	Integer  uint64
}

// Decode parses one complete protobuf wire message without assigning schema names.
func Decode(data []byte) ([]Field, error) {
	fields := make([]Field, 0)
	for offset := 0; offset < len(data); {
		tag, tagBytes := ReadVarint(data[offset:])
		if tagBytes == 0 || tag>>protobufFieldShift == 0 {
			return nil, fmt.Errorf("invalid protobuf tag at byte %d", offset)
		}
		offset += tagBytes
		field := Field{Number: int(tag >> protobufFieldShift), WireType: int(tag & protobufWireMask)}
		switch field.WireType {
		case varintWireType:
			value, valueBytes := ReadVarint(data[offset:])
			if valueBytes == 0 {
				return nil, fmt.Errorf("invalid varint at byte %d", offset)
			}
			field.Integer = value
			offset += valueBytes
		case fixed64WireType:
			if offset+fixed64ByteLength > len(data) {
				return nil, fmt.Errorf("truncated fixed64 at byte %d", offset)
			}
			field.Bytes = data[offset : offset+fixed64ByteLength]
			offset += fixed64ByteLength
		case lengthDelimitedWireType:
			length, lengthBytes := ReadVarint(data[offset:])
			if lengthBytes == 0 || length > uint64(len(data)-offset-lengthBytes) {
				return nil, fmt.Errorf("invalid length-delimited field at byte %d", offset)
			}
			offset += lengthBytes
			field.Bytes = data[offset : offset+int(length)]
			offset += int(length)
		case fixed32WireType:
			if offset+fixed32ByteLength > len(data) {
				return nil, fmt.Errorf("truncated fixed32 at byte %d", offset)
			}
			field.Bytes = data[offset : offset+fixed32ByteLength]
			offset += fixed32ByteLength
		default:
			return nil, fmt.Errorf("unsupported wire type %d", field.WireType)
		}
		fields = append(fields, field)
	}
	return fields, nil
}

// ReadVarint reads one protobuf varint and returns zero bytes when incomplete.
func ReadVarint(data []byte) (uint64, int) {
	var value uint64
	var shift uint
	for index, byteValue := range data {
		value |= uint64(byteValue&0x7f) << shift
		if byteValue&0x80 == 0 {
			return value, index + 1
		}
		shift += 7
		if shift >= maximumVarintShift {
			return 0, 0
		}
	}
	return 0, 0
}

// FirstBytes returns the first length-delimited field with the requested number.
func FirstBytes(fields []Field, number int) []byte {
	for _, field := range fields {
		if field.Number == number && field.WireType == lengthDelimitedWireType {
			return field.Bytes
		}
	}
	return nil
}

// Varint returns the first varint field with the requested number.
func Varint(fields []Field, number int) (uint64, bool) {
	for _, field := range fields {
		if field.Number == number && field.WireType == varintWireType {
			return field.Integer, true
		}
	}
	return 0, false
}
