package protocol

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

// Binary protocol constants
const (
	// Message types
	BinaryMsgTypePriceUpdate = 0x01
	BinaryMsgTypeOrderUpdate = 0x02
	BinaryMsgTypeHeartbeat   = 0x03
	BinaryMsgTypeError       = 0x04
	
	// Fixed message sizes
	BinaryPriceUpdateSize = 32  // 8 + 8 + 8 + 8 bytes
	BinaryOrderUpdateSize = 64  // Variable size, but max 64 bytes
	BinaryHeartbeatSize   = 16  // 8 + 8 bytes
	BinaryErrorSize       = 32  // 4 + 28 bytes
	
	// Header size
	BinaryHeaderSize = 4 // 1 byte type + 1 byte flags + 2 bytes length
)

// BinaryHeader represents the binary message header
type BinaryHeader struct {
	Type   uint8  // Message type
	Flags  uint8  // Message flags
	Length uint16 // Message length (excluding header)
}

// BinaryPriceUpdate represents a binary price update message
type BinaryPriceUpdate struct {
	Symbol    [8]byte // Fixed-size symbol (padded with zeros)
	Price     uint64  // Price as integer (scaled by 1e8)
	Volume    uint64  // Volume as integer (scaled by 1e8)
	Timestamp int64   // Unix nanoseconds
}

// BinaryOrderUpdate represents a binary order update message
type BinaryOrderUpdate struct {
	OrderID        [16]byte // Fixed-size order ID (UUID bytes)
	Symbol         [8]byte  // Fixed-size symbol
	Side           uint8    // 0 = buy, 1 = sell
	Status         uint8    // Order status
	FilledQuantity uint64   // Filled quantity (scaled by 1e8)
	AveragePrice   uint64   // Average price (scaled by 1e8)
	Timestamp      int64    // Unix nanoseconds
	Reserved       [14]byte // Reserved for future use
}

// BinaryHeartbeat represents a binary heartbeat message
type BinaryHeartbeat struct {
	Timestamp int64 // Unix nanoseconds
	Sequence  int64 // Sequence number
}

// BinaryError represents a binary error message
type BinaryError struct {
	Code      uint32   // Error code
	Message   [28]byte // Error message (fixed size)
}

// MarshalBinaryPriceUpdate marshals a price update to binary format
func MarshalBinaryPriceUpdate(symbol string, price, volume float64, timestamp time.Time) []byte {
	buf := make([]byte, BinaryHeaderSize+BinaryPriceUpdateSize)
	
	// Write header
	buf[0] = BinaryMsgTypePriceUpdate
	buf[1] = 0 // No flags
	binary.LittleEndian.PutUint16(buf[2:4], BinaryPriceUpdateSize)
	
	// Write price update
	update := BinaryPriceUpdate{
		Price:     uint64(price * 1e8),     // Scale to avoid floating point
		Volume:    uint64(volume * 1e8),    // Scale to avoid floating point
		Timestamp: timestamp.UnixNano(),
	}
	
	// Copy symbol (truncate or pad as needed)
	copy(update.Symbol[:], []byte(symbol))
	
	// Write to buffer
	offset := BinaryHeaderSize
	copy(buf[offset:offset+8], update.Symbol[:])
	binary.LittleEndian.PutUint64(buf[offset+8:offset+16], update.Price)
	binary.LittleEndian.PutUint64(buf[offset+16:offset+24], update.Volume)
	binary.LittleEndian.PutUint64(buf[offset+24:offset+32], uint64(update.Timestamp))
	
	return buf
}

// UnmarshalBinaryPriceUpdate unmarshals a binary price update
func UnmarshalBinaryPriceUpdate(data []byte) (*BinaryPriceUpdate, error) {
	if len(data) < BinaryHeaderSize+BinaryPriceUpdateSize {
		return nil, fmt.Errorf("insufficient data for price update")
	}
	
	// Verify header
	if data[0] != BinaryMsgTypePriceUpdate {
		return nil, fmt.Errorf("invalid message type: expected %d, got %d", BinaryMsgTypePriceUpdate, data[0])
	}
	
	length := binary.LittleEndian.Uint16(data[2:4])
	if length != BinaryPriceUpdateSize {
		return nil, fmt.Errorf("invalid message length: expected %d, got %d", BinaryPriceUpdateSize, length)
	}
	
	// Parse price update
	offset := BinaryHeaderSize
	update := &BinaryPriceUpdate{}
	
	copy(update.Symbol[:], data[offset:offset+8])
	update.Price = binary.LittleEndian.Uint64(data[offset+8 : offset+16])
	update.Volume = binary.LittleEndian.Uint64(data[offset+16 : offset+24])
	update.Timestamp = int64(binary.LittleEndian.Uint64(data[offset+24 : offset+32]))
	
	return update, nil
}

// MarshalBinaryOrderUpdate marshals an order update to binary format
func MarshalBinaryOrderUpdate(orderID, symbol string, side string, status uint8, filledQty, avgPrice float64, timestamp time.Time) []byte {
	buf := make([]byte, BinaryHeaderSize+BinaryOrderUpdateSize)
	
	// Write header
	buf[0] = BinaryMsgTypeOrderUpdate
	buf[1] = 0 // No flags
	binary.LittleEndian.PutUint16(buf[2:4], BinaryOrderUpdateSize)
	
	// Write order update
	update := BinaryOrderUpdate{
		Status:         status,
		FilledQuantity: uint64(filledQty * 1e8),
		AveragePrice:   uint64(avgPrice * 1e8),
		Timestamp:      timestamp.UnixNano(),
	}
	
	// Copy order ID (assume it's a UUID string)
	copy(update.OrderID[:], []byte(orderID))
	
	// Copy symbol
	copy(update.Symbol[:], []byte(symbol))
	
	// Set side
	if side == "sell" {
		update.Side = 1
	} else {
		update.Side = 0 // buy
	}
	
	// Write to buffer
	offset := BinaryHeaderSize
	copy(buf[offset:offset+16], update.OrderID[:])
	copy(buf[offset+16:offset+24], update.Symbol[:])
	buf[offset+24] = update.Side
	buf[offset+25] = update.Status
	binary.LittleEndian.PutUint64(buf[offset+26:offset+34], update.FilledQuantity)
	binary.LittleEndian.PutUint64(buf[offset+34:offset+42], update.AveragePrice)
	binary.LittleEndian.PutUint64(buf[offset+42:offset+50], uint64(update.Timestamp))
	// Reserved bytes are already zero
	
	return buf
}

// UnmarshalBinaryOrderUpdate unmarshals a binary order update
func UnmarshalBinaryOrderUpdate(data []byte) (*BinaryOrderUpdate, error) {
	if len(data) < BinaryHeaderSize+BinaryOrderUpdateSize {
		return nil, fmt.Errorf("insufficient data for order update")
	}
	
	// Verify header
	if data[0] != BinaryMsgTypeOrderUpdate {
		return nil, fmt.Errorf("invalid message type: expected %d, got %d", BinaryMsgTypeOrderUpdate, data[0])
	}
	
	length := binary.LittleEndian.Uint16(data[2:4])
	if length != BinaryOrderUpdateSize {
		return nil, fmt.Errorf("invalid message length: expected %d, got %d", BinaryOrderUpdateSize, length)
	}
	
	// Parse order update
	offset := BinaryHeaderSize
	update := &BinaryOrderUpdate{}
	
	copy(update.OrderID[:], data[offset:offset+16])
	copy(update.Symbol[:], data[offset+16:offset+24])
	update.Side = data[offset+24]
	update.Status = data[offset+25]
	update.FilledQuantity = binary.LittleEndian.Uint64(data[offset+26 : offset+34])
	update.AveragePrice = binary.LittleEndian.Uint64(data[offset+34 : offset+42])
	update.Timestamp = int64(binary.LittleEndian.Uint64(data[offset+42 : offset+50]))
	copy(update.Reserved[:], data[offset+50:offset+64])
	
	return update, nil
}

// MarshalBinaryHeartbeat marshals a heartbeat to binary format
func MarshalBinaryHeartbeat(timestamp time.Time, sequence int64) []byte {
	buf := make([]byte, BinaryHeaderSize+BinaryHeartbeatSize)
	
	// Write header
	buf[0] = BinaryMsgTypeHeartbeat
	buf[1] = 0 // No flags
	binary.LittleEndian.PutUint16(buf[2:4], BinaryHeartbeatSize)
	
	// Write heartbeat
	offset := BinaryHeaderSize
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(timestamp.UnixNano()))
	binary.LittleEndian.PutUint64(buf[offset+8:offset+16], uint64(sequence))
	
	return buf
}

// UnmarshalBinaryHeartbeat unmarshals a binary heartbeat
func UnmarshalBinaryHeartbeat(data []byte) (*BinaryHeartbeat, error) {
	if len(data) < BinaryHeaderSize+BinaryHeartbeatSize {
		return nil, fmt.Errorf("insufficient data for heartbeat")
	}
	
	// Verify header
	if data[0] != BinaryMsgTypeHeartbeat {
		return nil, fmt.Errorf("invalid message type: expected %d, got %d", BinaryMsgTypeHeartbeat, data[0])
	}
	
	length := binary.LittleEndian.Uint16(data[2:4])
	if length != BinaryHeartbeatSize {
		return nil, fmt.Errorf("invalid message length: expected %d, got %d", BinaryHeartbeatSize, length)
	}
	
	// Parse heartbeat
	offset := BinaryHeaderSize
	heartbeat := &BinaryHeartbeat{
		Timestamp: int64(binary.LittleEndian.Uint64(data[offset : offset+8])),
		Sequence:  int64(binary.LittleEndian.Uint64(data[offset+8 : offset+16])),
	}
	
	return heartbeat, nil
}

// MarshalBinaryError marshals an error to binary format
func MarshalBinaryError(code uint32, message string) []byte {
	buf := make([]byte, BinaryHeaderSize+BinaryErrorSize)
	
	// Write header
	buf[0] = BinaryMsgTypeError
	buf[1] = 0 // No flags
	binary.LittleEndian.PutUint16(buf[2:4], BinaryErrorSize)
	
	// Write error
	offset := BinaryHeaderSize
	binary.LittleEndian.PutUint32(buf[offset:offset+4], code)
	
	// Copy message (truncate if too long)
	msgBytes := []byte(message)
	if len(msgBytes) > 28 {
		msgBytes = msgBytes[:28]
	}
	copy(buf[offset+4:offset+32], msgBytes)
	
	return buf
}

// UnmarshalBinaryError unmarshals a binary error
func UnmarshalBinaryError(data []byte) (*BinaryError, error) {
	if len(data) < BinaryHeaderSize+BinaryErrorSize {
		return nil, fmt.Errorf("insufficient data for error")
	}
	
	// Verify header
	if data[0] != BinaryMsgTypeError {
		return nil, fmt.Errorf("invalid message type: expected %d, got %d", BinaryMsgTypeError, data[0])
	}
	
	length := binary.LittleEndian.Uint16(data[2:4])
	if length != BinaryErrorSize {
		return nil, fmt.Errorf("invalid message length: expected %d, got %d", BinaryErrorSize, length)
	}
	
	// Parse error
	offset := BinaryHeaderSize
	error := &BinaryError{
		Code: binary.LittleEndian.Uint32(data[offset : offset+4]),
	}
	copy(error.Message[:], data[offset+4:offset+32])
	
	return error, nil
}

// Helper functions for converting between binary and human-readable formats

// SymbolFromBytes converts a byte array to a symbol string
func SymbolFromBytes(bytes [8]byte) string {
	// Find the first null byte
	end := 8
	for i, b := range bytes {
		if b == 0 {
			end = i
			break
		}
	}
	return string(bytes[:end])
}

// OrderIDFromBytes converts a byte array to an order ID string
func OrderIDFromBytes(bytes [16]byte) string {
	// Find the first null byte
	end := 16
	for i, b := range bytes {
		if b == 0 {
			end = i
			break
		}
	}
	return string(bytes[:end])
}

// PriceFromScaled converts a scaled integer price to float64
func PriceFromScaled(scaled uint64) float64 {
	return float64(scaled) / 1e8
}

// VolumeFromScaled converts a scaled integer volume to float64
func VolumeFromScaled(scaled uint64) float64 {
	return float64(scaled) / 1e8
}

// SideToString converts a side byte to string
func SideToString(side uint8) string {
	if side == 1 {
		return "sell"
	}
	return "buy"
}

// StatusToString converts a status byte to string
func StatusToString(status uint8) string {
	switch status {
	case 0:
		return "pending"
	case 1:
		return "partial"
	case 2:
		return "filled"
	case 3:
		return "cancelled"
	case 4:
		return "rejected"
	default:
		return "unknown"
	}
}

// ErrorMessageFromBytes converts a byte array to an error message string
func ErrorMessageFromBytes(bytes [28]byte) string {
	// Find the first null byte
	end := 28
	for i, b := range bytes {
		if b == 0 {
			end = i
			break
		}
	}
	return string(bytes[:end])
}

// BinaryMessagePool provides pooling for binary message buffers
type BinaryMessagePool struct {
	priceUpdatePool *sync.Pool
	orderUpdatePool *sync.Pool
	heartbeatPool   *sync.Pool
	errorPool       *sync.Pool
}

// NewBinaryMessagePool creates a new binary message pool
func NewBinaryMessagePool() *BinaryMessagePool {
	return &BinaryMessagePool{
		priceUpdatePool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, BinaryHeaderSize+BinaryPriceUpdateSize)
			},
		},
		orderUpdatePool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, BinaryHeaderSize+BinaryOrderUpdateSize)
			},
		},
		heartbeatPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, BinaryHeaderSize+BinaryHeartbeatSize)
			},
		},
		errorPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, BinaryHeaderSize+BinaryErrorSize)
			},
		},
	}
}

// GetPriceUpdateBuffer gets a buffer for price update messages
func (p *BinaryMessagePool) GetPriceUpdateBuffer() []byte {
	return p.priceUpdatePool.Get().([]byte)
}

// PutPriceUpdateBuffer returns a buffer to the price update pool
func (p *BinaryMessagePool) PutPriceUpdateBuffer(buf []byte) {
	if len(buf) == BinaryHeaderSize+BinaryPriceUpdateSize {
		// Clear the buffer
		for i := range buf {
			buf[i] = 0
		}
		p.priceUpdatePool.Put(buf)
	}
}

// GetOrderUpdateBuffer gets a buffer for order update messages
func (p *BinaryMessagePool) GetOrderUpdateBuffer() []byte {
	return p.orderUpdatePool.Get().([]byte)
}

// PutOrderUpdateBuffer returns a buffer to the order update pool
func (p *BinaryMessagePool) PutOrderUpdateBuffer(buf []byte) {
	if len(buf) == BinaryHeaderSize+BinaryOrderUpdateSize {
		// Clear the buffer
		for i := range buf {
			buf[i] = 0
		}
		p.orderUpdatePool.Put(buf)
	}
}

// GetHeartbeatBuffer gets a buffer for heartbeat messages
func (p *BinaryMessagePool) GetHeartbeatBuffer() []byte {
	return p.heartbeatPool.Get().([]byte)
}

// PutHeartbeatBuffer returns a buffer to the heartbeat pool
func (p *BinaryMessagePool) PutHeartbeatBuffer(buf []byte) {
	if len(buf) == BinaryHeaderSize+BinaryHeartbeatSize {
		// Clear the buffer
		for i := range buf {
			buf[i] = 0
		}
		p.heartbeatPool.Put(buf)
	}
}

// GetErrorBuffer gets a buffer for error messages
func (p *BinaryMessagePool) GetErrorBuffer() []byte {
	return p.errorPool.Get().([]byte)
}

// PutErrorBuffer returns a buffer to the error pool
func (p *BinaryMessagePool) PutErrorBuffer(buf []byte) {
	if len(buf) == BinaryHeaderSize+BinaryErrorSize {
		// Clear the buffer
		for i := range buf {
			buf[i] = 0
		}
		p.errorPool.Put(buf)
	}
}
