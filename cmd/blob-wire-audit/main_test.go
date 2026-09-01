package main

import (
	"testing"

	"heimdall/internal/agent_adapters/antigravity/wire"
)

func TestWireDecode_Pos_DecodesNestedLengthDelimitedValue(t *testing.T) {
	fields, decodeErr := wire.Decode([]byte{0x08, 0x96, 0x01, 0x12, 0x03, 'c', 'a', 't'})

	if decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0].Number != 1 || fields[0].Integer != 150 {
		t.Fatalf("unexpected varint field: %#v", fields[0])
	}
	if fields[1].Number != 2 || string(fields[1].Bytes) != "cat" {
		t.Fatalf("unexpected bytes field: %#v", fields[1])
	}
}

func TestWireDecode_Neg_RejectsUnsupportedWireType(t *testing.T) {
	_, decodeErr := wire.Decode([]byte{0x0b})

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

func TestDecodeRawUsageProfile_Pos_SeparatesRawUsageAndContextScalars(t *testing.T) {
	data := []byte{
		0x0a, 0x10,
		0x22, 0x06, 0x08, 0x07, 0x10, 0x0b, 0x28, 0x0d,
		0x4a, 0x06, 0x52, 0x04, 0x08, 0x11, 0x20, 0x64,
	}

	profile, found := decodeRawUsageProfile(19, data)

	requireProfileEqual(t, found, true)
	requireProfileEqual(t, profile.generationIndex, 19)
	requireProfileEqual(t, profile.usageScalars[1], uint64(7))
	requireProfileEqual(t, profile.usageScalars[2], uint64(11))
	requireProfileEqual(t, profile.usageScalars[5], uint64(13))
	requireProfileEqual(t, profile.contextScalars[1], uint64(17))
	requireProfileEqual(t, profile.contextScalars[4], uint64(100))
}

func TestDecodeRawUsageProfile_Neg_RejectsMalformedWireData(t *testing.T) {
	profile, found := decodeRawUsageProfile(19, []byte{0x0b})

	requireProfileEqual(t, found, false)
	requireProfileEqual(t, profile.generationIndex, 0)
}

func requireProfileEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual %v, expected %v", actual, expected)
	}
}
