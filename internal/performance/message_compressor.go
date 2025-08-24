package performance

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/metrics"
	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
)

// CompressionAlgorithm defines the compression algorithm to use
type CompressionAlgorithm string

const (
	// CompressionNone uses no compression
	CompressionNone CompressionAlgorithm = "none"
	// CompressionGzip uses gzip compression
	CompressionGzip CompressionAlgorithm = "gzip"
	// CompressionZlib uses zlib compression
	CompressionZlib CompressionAlgorithm = "zlib"
	// CompressionDeflate uses deflate compression
	CompressionDeflate CompressionAlgorithm = "deflate"
	// CompressionZstd uses zstd compression
	CompressionZstd CompressionAlgorithm = "zstd"
)

// CompressionLevel defines the compression level to use
type CompressionLevel int

const (
	// CompressionLevelBestSpeed optimizes for speed
	CompressionLevelBestSpeed CompressionLevel = 1
	// CompressionLevelBalanced balances speed and compression
	CompressionLevelBalanced CompressionLevel = 5
	// CompressionLevelBestCompression optimizes for compression
	CompressionLevelBestCompression CompressionLevel = 9
)

// MessageCompressorConfig contains configuration for the message compressor
type MessageCompressorConfig struct {
	// DefaultAlgorithm is the default compression algorithm to use
	DefaultAlgorithm CompressionAlgorithm
	
	// DefaultLevel is the default compression level to use
	DefaultLevel CompressionLevel
	
	// MinSizeForCompression is the minimum size in bytes for compression
	MinSizeForCompression int
	
	// TypeSpecificAlgorithms defines specific algorithms for message types
	TypeSpecificAlgorithms map[string]CompressionAlgorithm
	
	// TypeSpecificLevels defines specific levels for message types
	TypeSpecificLevels map[string]CompressionLevel
	
	// EnableAdaptiveCompression enables adaptive compression based on message size
	EnableAdaptiveCompression bool
	
	// AdaptiveThresholds defines thresholds for adaptive compression
	AdaptiveThresholds []AdaptiveThreshold
	
	// EnableCompressorPool enables pooling of compressors
	EnableCompressorPool bool
	
	// PoolSize is the size of the compressor pool
	PoolSize int
}

// AdaptiveThreshold defines a threshold for adaptive compression
type AdaptiveThreshold struct {
	// SizeThreshold is the size threshold in bytes
	SizeThreshold int
	
	// Algorithm is the compression algorithm to use
	Algorithm CompressionAlgorithm
	
	// Level is the compression level to use
	Level CompressionLevel
}

// DefaultMessageCompressorConfig returns the default configuration
func DefaultMessageCompressorConfig() MessageCompressorConfig {
	return MessageCompressorConfig{
		DefaultAlgorithm:        CompressionZstd,
		DefaultLevel:            CompressionLevelBalanced,
		MinSizeForCompression:   256,
		TypeSpecificAlgorithms:  make(map[string]CompressionAlgorithm),
		TypeSpecificLevels:      make(map[string]CompressionLevel),
		EnableAdaptiveCompression: true,
		AdaptiveThresholds: []AdaptiveThreshold{
			{
				SizeThreshold: 1024,      // 1KB
				Algorithm:     CompressionZstd,
				Level:         CompressionLevelBestSpeed,
			},
			{
				SizeThreshold: 10 * 1024, // 10KB
				Algorithm:     CompressionZstd,
				Level:         CompressionLevelBalanced,
			},
			{
				SizeThreshold: 100 * 1024, // 100KB
				Algorithm:     CompressionZstd,
				Level:         CompressionLevelBestCompression,
			},
		},
		EnableCompressorPool: true,
		PoolSize:             10,
	}
}

// MessageCompressor compresses messages for efficient transmission
type MessageCompressor struct {
	// Configuration
	config MessageCompressorConfig
	
	// Compressor pools
	gzipPool    sync.Pool
	zlibPool    sync.Pool
	deflatePool sync.Pool
	zstdPool    sync.Pool
	
	// Logger
	logger *zap.Logger
	
	// Metrics
	metrics *metrics.WebSocketMetrics
}

// NewMessageCompressor creates a new message compressor
func NewMessageCompressor(
	config MessageCompressorConfig,
	logger *zap.Logger,
	metrics *metrics.WebSocketMetrics,
) *MessageCompressor {
	compressor := &MessageCompressor{
		config:  config,
		logger:  logger,
		metrics: metrics,
	}
	
	// Initialize compressor pools if enabled
	if config.EnableCompressorPool {
		compressor.initPools()
	}
	
	return compressor
}

// initPools initializes the compressor pools
func (c *MessageCompressor) initPools() {
	// Initialize gzip pool
	c.gzipPool = sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, int(c.config.DefaultLevel))
			return w
		},
	}
	
	// Initialize zlib pool
	c.zlibPool = sync.Pool{
		New: func() interface{} {
			w, _ := zlib.NewWriterLevel(nil, int(c.config.DefaultLevel))
			return w
		},
	}
	
	// Initialize deflate pool
	c.deflatePool = sync.Pool{
		New: func() interface{} {
			w, _ := flate.NewWriter(nil, int(c.config.DefaultLevel))
			return w
		},
	}
	
	// Initialize zstd pool
	c.zstdPool = sync.Pool{
		New: func() interface{} {
			w, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevel(c.config.DefaultLevel)))
			return w
		},
	}
}

// CompressMessage compresses a message
func (c *MessageCompressor) CompressMessage(message []byte, messageType string) ([]byte, error) {
	// Check if the message is large enough to compress
	if len(message) < c.config.MinSizeForCompression {
		return message, nil
	}
	
	// Determine the compression algorithm and level to use
	algorithm, level := c.getCompressionParams(message, messageType)
	
	// If no compression, return the original message
	if algorithm == CompressionNone {
		return message, nil
	}
	
	// Compress the message
	startTime := time.Now()
	compressed, err := c.compress(message, algorithm, level)
	duration := time.Since(startTime)
	
	// Record compression metrics
	if c.metrics != nil {
		c.metrics.RecordCompression(len(message), len(compressed), duration)
	}
	
	if err != nil {
		c.logger.Error("Failed to compress message",
			zap.Error(err),
			zap.String("algorithm", string(algorithm)),
			zap.Int("level", int(level)),
			zap.Int("original_size", len(message)),
			zap.String("message_type", messageType))
		return message, err
	}
	
	// If compression didn't help, return the original message
	if len(compressed) >= len(message) {
		return message, nil
	}
	
	c.logger.Debug("Compressed message",
		zap.String("algorithm", string(algorithm)),
		zap.Int("level", int(level)),
		zap.Int("original_size", len(message)),
		zap.Int("compressed_size", len(compressed)),
		zap.Float64("ratio", float64(len(message))/float64(len(compressed))),
		zap.String("message_type", messageType))
	
	return compressed, nil
}

// getCompressionParams determines the compression algorithm and level to use
func (c *MessageCompressor) getCompressionParams(message []byte, messageType string) (CompressionAlgorithm, CompressionLevel) {
	// Check for type-specific algorithm
	algorithm, ok := c.config.TypeSpecificAlgorithms[messageType]
	if !ok {
		algorithm = c.config.DefaultAlgorithm
	}
	
	// Check for type-specific level
	level, ok := c.config.TypeSpecificLevels[messageType]
	if !ok {
		level = c.config.DefaultLevel
	}
	
	// If adaptive compression is enabled, adjust based on message size
	if c.config.EnableAdaptiveCompression {
		size := len(message)
		for _, threshold := range c.config.AdaptiveThresholds {
			if size >= threshold.SizeThreshold {
				algorithm = threshold.Algorithm
				level = threshold.Level
			} else {
				break
			}
		}
	}
	
	return algorithm, level
}

// compress compresses a message using the specified algorithm and level
func (c *MessageCompressor) compress(message []byte, algorithm CompressionAlgorithm, level CompressionLevel) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	
	switch algorithm {
	case CompressionGzip:
		err = c.compressGzip(&buf, message, level)
	case CompressionZlib:
		err = c.compressZlib(&buf, message, level)
	case CompressionDeflate:
		err = c.compressDeflate(&buf, message, level)
	case CompressionZstd:
		err = c.compressZstd(&buf, message, level)
	default:
		return message, nil
	}
	
	if err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// compressGzip compresses a message using gzip
func (c *MessageCompressor) compressGzip(buf *bytes.Buffer, message []byte, level CompressionLevel) error {
	var w *gzip.Writer
	
	if c.config.EnableCompressorPool {
		w = c.gzipPool.Get().(*gzip.Writer)
		defer c.gzipPool.Put(w)
		w.Reset(buf)
	} else {
		var err error
		w, err = gzip.NewWriterLevel(buf, int(level))
		if err != nil {
			return err
		}
	}
	
	if _, err := w.Write(message); err != nil {
		return err
	}
	
	return w.Close()
}

// compressZlib compresses a message using zlib
func (c *MessageCompressor) compressZlib(buf *bytes.Buffer, message []byte, level CompressionLevel) error {
	var w *zlib.Writer
	
	if c.config.EnableCompressorPool {
		w = c.zlibPool.Get().(*zlib.Writer)
		defer c.zlibPool.Put(w)
		w.Reset(buf)
	} else {
		var err error
		w, err = zlib.NewWriterLevel(buf, int(level))
		if err != nil {
			return err
		}
	}
	
	if _, err := w.Write(message); err != nil {
		return err
	}
	
	return w.Close()
}

// compressDeflate compresses a message using deflate
func (c *MessageCompressor) compressDeflate(buf *bytes.Buffer, message []byte, level CompressionLevel) error {
	var w *flate.Writer
	
	if c.config.EnableCompressorPool {
		w = c.deflatePool.Get().(*flate.Writer)
		defer c.deflatePool.Put(w)
		w.Reset(buf)
	} else {
		var err error
		w, err = flate.NewWriter(buf, int(level))
		if err != nil {
			return err
		}
	}
	
	if _, err := w.Write(message); err != nil {
		return err
	}
	
	return w.Close()
}

// compressZstd compresses a message using zstd
func (c *MessageCompressor) compressZstd(buf *bytes.Buffer, message []byte, level CompressionLevel) error {
	var w *zstd.Encoder
	
	if c.config.EnableCompressorPool {
		w = c.zstdPool.Get().(*zstd.Encoder)
		defer c.zstdPool.Put(w)
		w.Reset(buf)
	} else {
		var err error
		w, err = zstd.NewWriter(buf, zstd.WithEncoderLevel(zstd.EncoderLevel(level)))
		if err != nil {
			return err
		}
	}
	
	if _, err := w.Write(message); err != nil {
		return err
	}
	
	return w.Close()
}

// CompressJSON compresses a JSON object
func (c *MessageCompressor) CompressJSON(v interface{}, messageType string) ([]byte, error) {
	// Marshal the object to JSON
	jsonData, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	
	// Compress the JSON data
	return c.CompressMessage(jsonData, messageType)
}

// DecompressMessage decompresses a message
func (c *MessageCompressor) DecompressMessage(message []byte, algorithm CompressionAlgorithm) ([]byte, error) {
	// If no compression, return the original message
	if algorithm == CompressionNone {
		return message, nil
	}
	
	// Decompress the message
	startTime := time.Now()
	decompressed, err := c.decompress(message, algorithm)
	duration := time.Since(startTime)
	
	// Record decompression metrics (using the same method as compression)
	if c.metrics != nil && err == nil {
		c.metrics.RecordCompression(len(decompressed), len(message), duration)
	}
	
	if err != nil {
		c.logger.Error("Failed to decompress message",
			zap.Error(err),
			zap.String("algorithm", string(algorithm)),
			zap.Int("compressed_size", len(message)))
		return nil, err
	}
	
	return decompressed, nil
}

// decompress decompresses a message using the specified algorithm
func (c *MessageCompressor) decompress(message []byte, algorithm CompressionAlgorithm) ([]byte, error) {
	var buf bytes.Buffer
	var reader io.Reader
	
	// Create the appropriate reader
	switch algorithm {
	case CompressionGzip:
		r, err := gzip.NewReader(bytes.NewReader(message))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		reader = r
		
	case CompressionZlib:
		r, err := zlib.NewReader(bytes.NewReader(message))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		reader = r
		
	case CompressionDeflate:
		reader = flate.NewReader(bytes.NewReader(message))
		defer reader.(io.Closer).Close()
		
	case CompressionZstd:
		r, err := zstd.NewReader(bytes.NewReader(message))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		reader = r
		
	default:
		return message, nil
	}
	
	// Read the decompressed data
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// DecompressJSON decompresses a JSON message
func (c *MessageCompressor) DecompressJSON(message []byte, algorithm CompressionAlgorithm, v interface{}) error {
	// Decompress the message
	decompressed, err := c.DecompressMessage(message, algorithm)
	if err != nil {
		return err
	}
	
	// Unmarshal the JSON data
	return json.Unmarshal(decompressed, v)
}

