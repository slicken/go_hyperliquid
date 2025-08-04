package hyperliquid

import (
	"bytes"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

// Global JSON instances for maximum performance
var (
	// Fast JSON instance - fastest for most use cases
	fastJSON = jsoniter.ConfigFastest
	// Compatible JSON instance - for compatibility when needed
	compatibleJSON = jsoniter.ConfigCompatibleWithStandardLibrary
)

// JSON pools for high-frequency reuse
var (
	// Buffer pool for JSON marshaling
	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	// Encoder pool for repeated marshaling
	encoderPool = sync.Pool{
		New: func() interface{} {
			return fastJSON.NewEncoder(nil)
		},
	}

	// Decoder pool for repeated unmarshaling
	decoderPool = sync.Pool{
		New: func() interface{} {
			return fastJSON.NewDecoder(nil)
		},
	}
)

// FastMarshal marshals an object to JSON using the fastest configuration
// This is optimized for high-frequency trading scenarios
func FastMarshal(v interface{}) ([]byte, error) {
	return fastJSON.Marshal(v)
}

// FastUnmarshal unmarshals JSON to an object using the fastest configuration
func FastUnmarshal(data []byte, v interface{}) error {
	return fastJSON.Unmarshal(data, v)
}

// FastMarshalToString marshals to string (useful for logging)
func FastMarshalToString(v interface{}) (string, error) {
	return fastJSON.MarshalToString(v)
}

// FastUnmarshalFromString unmarshals from string
func FastUnmarshalFromString(str string, v interface{}) error {
	return fastJSON.UnmarshalFromString(str, v)
}

// PooledMarshal marshals using a pooled buffer and encoder for maximum efficiency
// This is the fastest method for repeated marshaling of similar objects
func PooledMarshal(v interface{}) ([]byte, error) {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	encoder := encoderPool.Get().(*jsoniter.Encoder)
	// Reset encoder to use our buffer
	*encoder = *fastJSON.NewEncoder(buf)
	defer encoderPool.Put(encoder)

	err := encoder.Encode(v)
	if err != nil {
		return nil, err
	}

	// Copy the result to avoid buffer reuse issues
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// PooledUnmarshal unmarshals using a pooled decoder
func PooledUnmarshal(data []byte, v interface{}) error {
	decoder := decoderPool.Get().(*jsoniter.Decoder)
	defer decoderPool.Put(decoder)

	// Reset decoder to use our data
	*decoder = *fastJSON.NewDecoder(bytes.NewReader(data))
	return decoder.Decode(v)
}

// CompatibleMarshal uses standard library compatible marshaling
// Use this when you need exact compatibility with standard library
func CompatibleMarshal(v interface{}) ([]byte, error) {
	return compatibleJSON.Marshal(v)
}

// CompatibleUnmarshal uses standard library compatible unmarshaling
func CompatibleUnmarshal(data []byte, v interface{}) error {
	return compatibleJSON.Unmarshal(data, v)
}

// PreMarshaledMessage represents a pre-marshaled JSON message
// This is the fastest approach for static messages
type PreMarshaledMessage struct {
	data []byte
}

// NewPreMarshaledMessage creates a pre-marshaled message
func NewPreMarshaledMessage(v interface{}) (*PreMarshaledMessage, error) {
	data, err := FastMarshal(v)
	if err != nil {
		return nil, err
	}
	return &PreMarshaledMessage{data: data}, nil
}

// Bytes returns the marshaled bytes
func (p *PreMarshaledMessage) Bytes() []byte {
	return p.data
}

// String returns the marshaled string
func (p *PreMarshaledMessage) String() string {
	return string(p.data)
}
