package wire

import "testing"

const (
	validVarintValue    = 300
	validVarintByteSize = 2
)

func TestReadVarint(t *testing.T) {
	testCases := []struct {
		name          string
		input         []byte
		expectedValue uint64
		expectedBytes int
	}{
		{name: "single byte", input: []byte{0x01}, expectedValue: 1, expectedBytes: 1},
		{name: "multi byte", input: []byte{0xac, 0x02}, expectedValue: validVarintValue, expectedBytes: validVarintByteSize},
		{name: "truncated", input: []byte{0x80}, expectedValue: 0, expectedBytes: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			value, byteCount := ReadVarint(testCase.input)
			requireEqual(t, value, testCase.expectedValue)
			requireEqual(t, byteCount, testCase.expectedBytes)
		})
	}
}

func TestDecode(t *testing.T) {
	fields, err := Decode([]byte{0x08, 0x2a, 0x12, 0x02, 0x68, 0x69})

	requireNoError(t, err)
	requireEqual(t, len(fields), 2)
	requireEqual(t, fields[0].Number, 1)
	requireEqual(t, fields[0].Integer, uint64(42))
	requireEqual(t, string(FirstBytes(fields, 2)), "hi")
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireEqual[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("actual %v, expected %v", actual, expected)
	}
}
