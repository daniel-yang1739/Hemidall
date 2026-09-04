package schema

import (
	"fmt"
	"time"

	"heimdall/internal/agent_adapters/antigravity/wire"
)

// MessageRole represents the conversational role enum defined in cortex.proto (MessagePromptMetadata.role).
type MessageRole int64

const (
	RoleUnspecified MessageRole = 0
	RoleUser        MessageRole = 1 // Client inbound prompt, user request, or context summary
	RoleAssistant   MessageRole = 2 // Model response turn or tool invocation
	RoleSystem      MessageRole = 3 // System constitutional instructions
	RoleToolResult  MessageRole = 4 // Local tool execution output / function response
)

// String returns the canonical role name for telemetry display.
func (r MessageRole) String() string {
	switch r {
	case RoleUser:
		return "USER"
	case RoleAssistant:
		return "ASSISTANT"
	case RoleSystem:
		return "SYSTEM"
	case RoleToolResult:
		return "TOOL_RESULT"
	default:
		return fmt.Sprintf("UNKNOWN_%d", r)
	}
}

// Description returns a human-friendly description of the role.
func (r MessageRole) Description() string {
	switch r {
	case RoleUser:
		return "User Request / Context Summary"
	case RoleAssistant:
		return "Model Turn / Tool Call"
	case RoleSystem:
		return "System Prompt"
	case RoleToolResult:
		return "Tool Execution Output"
	default:
		return "Unspecified Role"
	}
}

// RootEnvelope represents the outer container stored in SQLite gen_metadata.
type RootEnvelope struct {
	ChatModelData []byte
	ExecutionID   string
	Status        int64
}

// ChatModelMetadata represents exa.cortex_pb.ChatModelMetadata (Field 1).
type ChatModelMetadata struct {
	SystemPrompt      string
	MessagePrompts    []MessagePromptMetadata
	ModelEnum         int64
	Usage             ModelUsage
	Tools             []ToolDefinition
	ChatStart         ChatStartMetadata
	TimeToFirstToken  time.Duration
	StreamingDuration time.Duration
	ResponseModel     string
	CustomMetadata    map[string]string
}

// ModelUsage represents exa.cortex_pb.ModelUsage (Field 4).
type ModelUsage struct {
	ModelCode            int64
	UncachedPromptTokens int64
	ThinkingOutputTokens int64
	CachedContentTokens  int64
	ProviderTier         int64
	BotID                string
	OutputContentTokens  int64
	UpstreamRequestID    string
}

// ChatStartMetadata represents exa.cortex_pb.ChatStartMetadata (Field 9).
type ChatStartMetadata struct {
	ObservedContextTokens int64
	ContextLimitTokens    int64
	StartStepIndex        int64
	CheckpointIndex       int64
}

// MessagePromptMetadata represents exa.cortex_pb.MessagePromptMetadata (Field 2).
type MessagePromptMetadata struct {
	Role          MessageRole
	Content       string
	SequenceIndex int64
	RawByteSize   int
}

// ToolDefinition represents exa.cortex_pb.ToolDefinition (Field 8).
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema string
}

// DecodeRootEnvelope extracts the ChatModelMetadata payload and envelope metadata.
func DecodeRootEnvelope(data []byte) (RootEnvelope, error) {
	fields, err := wire.Decode(data)
	if err != nil {
		return RootEnvelope{}, fmt.Errorf("decode root envelope: %w", err)
	}
	var env RootEnvelope
	for _, f := range fields {
		switch f.Number {
		case 1:
			env.ChatModelData = f.Bytes
		case 4:
			env.ExecutionID = string(f.Bytes)
		case 10:
			env.Status = int64(f.Integer)
		}
	}
	return env, nil
}

// DecodeChatModelMetadata parses the complete ChatModelMetadata message.
func DecodeChatModelMetadata(data []byte) (ChatModelMetadata, error) {
	fields, err := wire.Decode(data)
	if err != nil {
		return ChatModelMetadata{}, fmt.Errorf("decode ChatModelMetadata: %w", err)
	}

	var meta ChatModelMetadata
	meta.CustomMetadata = make(map[string]string)

	for _, f := range fields {
		switch f.Number {
		case 1:
			meta.SystemPrompt = string(f.Bytes)
		case 2:
			prompt, pErr := DecodeMessagePrompt(f.Bytes)
			if pErr == nil {
				meta.MessagePrompts = append(meta.MessagePrompts, prompt)
			}
		case 3:
			meta.ModelEnum = int64(f.Integer)
		case 4:
			usage, uErr := DecodeModelUsage(f.Bytes)
			if uErr == nil {
				meta.Usage = usage
			}
		case 8:
			tool, tErr := DecodeToolDefinition(f.Bytes)
			if tErr == nil {
				meta.Tools = append(meta.Tools, tool)
			}
		case 9:
			chatStart, csErr := DecodeChatStart(f.Bytes)
			if csErr == nil {
				meta.ChatStart = chatStart
			}
		case 11:
			meta.TimeToFirstToken = decodeDuration(f.Bytes)
		case 12:
			meta.StreamingDuration = decodeDuration(f.Bytes)
		case 17:
			if meta.Usage.UncachedPromptTokens == 0 && meta.Usage.CachedContentTokens == 0 {
				if subFields, sErr := wire.Decode(f.Bytes); sErr == nil {
					for _, sf := range subFields {
						if sf.Number == 2 {
							if usage, uErr := DecodeModelUsage(sf.Bytes); uErr == nil {
								meta.Usage = usage
							}
						}
					}
				}
			}
		case 19:
			meta.ResponseModel = string(f.Bytes)
		case 20:
			k, v := decodeMapEntry(f.Bytes)
			if k != "" {
				meta.CustomMetadata[k] = v
			}
		}
	}

	return meta, nil
}

// DecodeMessagePrompt parses a MessagePromptMetadata turn.
func DecodeMessagePrompt(data []byte) (MessagePromptMetadata, error) {
	fields, err := wire.Decode(data)
	if err != nil {
		return MessagePromptMetadata{}, err
	}
	prompt := MessagePromptMetadata{RawByteSize: len(data)}
	for _, f := range fields {
		switch f.Number {
		case 2:
			prompt.Role = MessageRole(f.Integer)
		case 3:
			prompt.Content = string(f.Bytes)
		case 18:
			prompt.SequenceIndex = int64(f.Integer)
		}
	}
	return prompt, nil
}

// DecodeModelUsage parses the upstream token billing metadata.
func DecodeModelUsage(data []byte) (ModelUsage, error) {
	fields, err := wire.Decode(data)
	if err != nil {
		return ModelUsage{}, err
	}
	var usage ModelUsage
	for _, f := range fields {
		switch f.Number {
		case 1:
			usage.ModelCode = int64(f.Integer)
		case 2:
			usage.UncachedPromptTokens = int64(f.Integer)
		case 3:
			usage.ThinkingOutputTokens = int64(f.Integer)
		case 5:
			usage.CachedContentTokens = int64(f.Integer)
		case 6:
			usage.ProviderTier = int64(f.Integer)
		case 7:
			usage.BotID = string(f.Bytes)
		case 9:
			usage.OutputContentTokens = int64(f.Integer)
		case 11:
			usage.UpstreamRequestID = string(f.Bytes)
		}
	}
	return usage, nil
}

// DecodeChatStart parses ChatStartMetadata.
func DecodeChatStart(data []byte) (ChatStartMetadata, error) {
	fields, err := wire.Decode(data)
	if err != nil {
		return ChatStartMetadata{}, err
	}
	var meta ChatStartMetadata
	for _, f := range fields {
		switch f.Number {
		case 10:
			// Nested ChatStartMetadata is inside sub-field 10
			subFields, sErr := wire.Decode(f.Bytes)
			if sErr == nil {
				for _, sf := range subFields {
					switch sf.Number {
					case 1:
						meta.ObservedContextTokens = int64(sf.Integer)
					case 4:
						meta.ContextLimitTokens = int64(sf.Integer)
					case 5:
						meta.StartStepIndex = int64(sf.Integer)
					case 6:
						meta.CheckpointIndex = int64(sf.Integer)
					}
				}
			}
		}
	}
	return meta, nil
}

// DecodeToolDefinition parses a native tool declaration.
func DecodeToolDefinition(data []byte) (ToolDefinition, error) {
	fields, err := wire.Decode(data)
	if err != nil {
		return ToolDefinition{}, err
	}
	var tool ToolDefinition
	for _, f := range fields {
		switch f.Number {
		case 1:
			tool.Name = string(f.Bytes)
		case 2:
			tool.Description = string(f.Bytes)
		case 3:
			tool.InputSchema = string(f.Bytes)
		}
	}
	return tool, nil
}

func decodeDuration(data []byte) time.Duration {
	fields, err := wire.Decode(data)
	if err != nil {
		return 0
	}
	sec, _ := wire.Varint(fields, 1)
	nanos, _ := wire.Varint(fields, 2)
	return time.Duration(sec)*time.Second + time.Duration(nanos)*time.Nanosecond
}

func decodeMapEntry(data []byte) (string, string) {
	fields, err := wire.Decode(data)
	if err != nil {
		return "", ""
	}
	key := string(wire.FirstBytes(fields, 1))
	val := string(wire.FirstBytes(fields, 2))
	return key, val
}
