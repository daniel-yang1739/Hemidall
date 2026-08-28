package antigravity

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
)

// GeminiGenerationMetadata contains telemetry extracted from Google Gemini Protobuf
type GeminiGenerationMetadata struct {
	GenIndex     int
	LastStepIdx  int
	TotalTokens  int
	CachedTokens int
	ContextLimit int
	CacheHitRate float64
	ModelName    string
}

// ParseGeminiGenMetadata decodes the raw Protobuf BLOB from the gen_metadata table
func ParseGeminiGenMetadata(genIndex int, data []byte) (*GeminiGenerationMetadata, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty protobuf data")
	}

	meta := &GeminiGenerationMetadata{
		GenIndex:     genIndex,
		ContextLimit: 256000,
	}

	// Extract LastStepIndex from raw byte patterns if present
	reStep := regexp.MustCompile(`last_step_index\x12\x04(\d+)`)
	if match := reStep.FindSubmatch(data); len(match) > 1 {
		if s, err := strconv.Atoi(string(match[1])); err == nil {
			meta.LastStepIdx = s
		}
	}

	// Extract ModelName
	reModel := regexp.MustCompile(`(gemini-[a-zA-Z0-9\.\-]+)`)
	if match := reModel.FindSubmatch(data); len(match) > 1 {
		meta.ModelName = string(match[1])
	}

	// Parse Protobuf hierarchy: Field 1 -> Field 9 -> Field 10
	msg := parseProtoMessage(data)
	if f1, ok := msg[1].(map[int]interface{}); ok {
		if f9, ok := f1[9].(map[int]interface{}); ok {
			if f10, ok := f9[10].(map[int]interface{}); ok {
				// Field 1: Total prompt tokens
				if tot, ok := f10[1].(uint64); ok {
					meta.TotalTokens = int(tot)
				}

				// Field 4: Context window limit (e.g. 256,000)
				if lim, ok := f10[4].(uint64); ok && lim > 0 {
					meta.ContextLimit = int(lim)
				}

				// Field 3: Cache details
				if f3, ok := f10[3].(map[int]interface{}); ok {
					if f3_f1, ok := f3[1].(map[int]interface{}); ok {
						// Field 4: Cached content tokens
						if cached, ok := f3_f1[4].(uint64); ok {
							meta.CachedTokens = int(cached)
						}
					}
				}
			}
		}
	}

	if meta.TotalTokens > 0 {
		meta.CacheHitRate = float64(meta.CachedTokens) / float64(meta.TotalTokens) * 100.0
	}

	return meta, nil
}

func parseProtoMessage(buf []byte) map[int]interface{} {
	m := make(map[int]interface{})
	p := 0
	for p < len(buf) {
		tagWire, n := readVarint(buf[p:])
		if n <= 0 {
			break
		}
		p += n

		fieldNum := int(tagWire >> 3)
		wireType := int(tagWire & 0x7)
		if fieldNum == 0 {
			break
		}

		switch wireType {
		case 0: // Varint
			val, n := readVarint(buf[p:])
			if n <= 0 {
				return m
			}
			p += n
			m[fieldNum] = val

		case 2: // Length-delimited (sub-message, string, bytes)
			length, n := readVarint(buf[p:])
			if n <= 0 {
				return m
			}
			p += n
			if p+int(length) > len(buf) {
				return m
			}
			subBuf := buf[p : p+int(length)]
			p += int(length)

			// Recursively parse as sub-message
			subMsg := parseProtoMessage(subBuf)
			if len(subMsg) > 0 {
				m[fieldNum] = subMsg
			} else {
				// Keep raw bytes or string
				if isPrintable(subBuf) {
					m[fieldNum] = string(subBuf)
				} else {
					m[fieldNum] = subBuf
				}
			}

		case 1: // 64-bit
			p += 8
		case 5: // 32-bit
			p += 4
		default:
			return m
		}
	}
	return m
}

func readVarint(buf []byte) (uint64, int) {
	var val uint64
	var shift uint
	for i, b := range buf {
		val |= uint64(b&0x7f) << shift
		if (b & 0x80) == 0 {
			return val, i + 1
		}
		shift += 7
		if shift >= 64 {
			return 0, 0
		}
	}
	return 0, 0
}

func isPrintable(buf []byte) bool {
	if len(buf) == 0 {
		return false
	}
	for _, b := range buf {
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			return false
		}
		if b > 126 {
			return false
		}
	}
	return bytes.TrimSpace(buf) != nil
}
