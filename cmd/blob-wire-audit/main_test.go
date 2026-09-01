package main

import "testing"

func TestDecodeFields_Pos_DecodesNestedLengthDelimitedValue(t *testing.T) {
	fields, decodeErr := decodeFields([]byte{0x08, 0x96, 0x01, 0x12, 0x03, 'c', 'a', 't'})

	if decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0].number != 1 || fields[0].integer != 150 {
		t.Fatalf("unexpected varint field: %#v", fields[0])
	}
	if fields[1].number != 2 || string(fields[1].bytes) != "cat" {
		t.Fatalf("unexpected bytes field: %#v", fields[1])
	}
}

func TestDecodeFields_Neg_RejectsUnsupportedWireType(t *testing.T) {
	_, decodeErr := decodeFields([]byte{0x0b})

	if decodeErr == nil {
		t.Fatal("expected unsupported wire type error")
	}
}

func TestPrintableText_Pos_RejectsBinaryValues(t *testing.T) {
	if printableText([]byte{'o', 'k'}) != "ok" {
		t.Fatal("expected printable text")
	}
	if printableText([]byte{'o', 0x00, 'k'}) != "" {
		t.Fatal("expected binary data to be rejected")
	}
}
