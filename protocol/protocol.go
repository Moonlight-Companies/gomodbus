package protocol

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/Moonlight-Companies/gomodbus/common"
	"github.com/Moonlight-Companies/gomodbus/logging"
)

// ProtocolHandler implements the common.Protocol interface for Modbus protocol
type ProtocolHandler struct {
	logger common.LoggerInterface
}

// Option is a function that configures a ProtocolHandler
type Option func(*ProtocolHandler)

// WithLogger sets the logger for the protocol handler
func WithLogger(logger common.LoggerInterface) Option {
	return func(p *ProtocolHandler) {
		p.logger = logger
	}
}

// NewProtocolHandler creates a new ProtocolHandler with options
func NewProtocolHandler(options ...Option) *ProtocolHandler {
	handler := &ProtocolHandler{
		logger: logging.NewLogger(), // Default logger
	}

	// Apply options
	for _, option := range options {
		option(handler)
	}

	return handler
}

// WithLogger returns a new ProtocolHandler with the given logger
func (h *ProtocolHandler) WithLogger(logger common.LoggerInterface) common.Protocol {
	return NewProtocolHandler(WithLogger(logger))
}

// generateReadRequest is a helper function for generating read requests that follow the same pattern
// (read coils, read discrete inputs, read holding registers, read input registers)
func (h *ProtocolHandler) generateReadRequest(itemType string, address common.Address, quantity common.Quantity, maxQuantity common.Quantity) ([]byte, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Generating read %s request: address=%d, quantity=%d", itemType, address, quantity)

	if quantity == 0 || quantity > maxQuantity {
		h.logger.Error(ctx, "Invalid quantity for read %s request: %d (max %d)", itemType, quantity, maxQuantity)
		return nil, common.ErrInvalidQuantity
	}

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], uint16(address))
	binary.BigEndian.PutUint16(data[2:4], uint16(quantity))

	h.logger.Debug(ctx, "Generated read %s request data: %v", itemType, data)
	return data, nil
}

// parseBitResponse is a helper function for parsing responses that contain bit values
// (coils and discrete inputs)
func (h *ProtocolHandler) parseBitResponse(itemType string, data []byte, quantity common.Quantity) ([]bool, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Parsing read %s response: data=%v, quantity=%d", itemType, data, quantity)

	if len(data) == 0 {
		h.logger.Error(ctx, "Empty response for read %s", itemType)
		return nil, common.ErrEmptyResponse
	}

	// First byte is the byte count
	byteCount := int(data[0])
	if len(data) != byteCount+1 {
		h.logger.Error(ctx, "Invalid response length for read %s: expected %d, got %d",
			itemType, byteCount+1, len(data))
		return nil, common.ErrInvalidResponseLength
	}

	// Calculate the expected byte count
	expectedByteCount := int(math.Ceil(float64(quantity) / 8.0))
	if byteCount != expectedByteCount {
		h.logger.Error(ctx, "Invalid byte count for read %s: expected %d, got %d",
			itemType, expectedByteCount, byteCount)
		return nil, common.ErrInvalidResponseLength
	}

	// Parse the values
	values := make([]bool, quantity)
	for i := 0; i < int(quantity); i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		byteValue := data[1+byteIndex]
		values[i] = ((byteValue >> uint(bitIndex)) & 0x01) == 1
	}

	h.logger.Debug(ctx, "Parsed %d %s values", len(values), itemType)
	return values, nil
}

// parseRegisterResponse is a helper function for parsing responses that contain register values
// (holding registers and input registers)
func (h *ProtocolHandler) parseRegisterResponse(itemType string, data []byte, quantity common.Quantity) ([]uint16, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Parsing read %s response: data=%v, quantity=%d", itemType, data, quantity)

	if len(data) == 0 {
		h.logger.Error(ctx, "Empty response for read %s", itemType)
		return nil, common.ErrEmptyResponse
	}

	// First byte is the byte count
	byteCount := int(data[0])
	if len(data) != byteCount+1 {
		h.logger.Error(ctx, "Invalid response length for read %s: expected %d, got %d",
			itemType, byteCount+1, len(data))
		return nil, common.ErrInvalidResponseLength
	}

	// Calculate the expected byte count
	expectedByteCount := int(quantity) * 2
	if byteCount != expectedByteCount {
		h.logger.Error(ctx, "Invalid byte count for read %s: expected %d, got %d",
			itemType, expectedByteCount, byteCount)
		return nil, common.ErrInvalidResponseLength
	}

	// Parse the values
	values := make([]uint16, quantity)
	for i := 0; i < int(quantity); i++ {
		values[i] = binary.BigEndian.Uint16(data[1+i*2 : 1+i*2+2])
	}

	h.logger.Debug(ctx, "Parsed %d %s values", len(values), itemType)
	return values, nil
}

// GenerateReadCoilsRequest generates a request to read coils
func (h *ProtocolHandler) GenerateReadCoilsRequest(address common.Address, quantity common.Quantity) ([]byte, error) {
	return h.generateReadRequest("coils", address, quantity, common.MaxCoilCount)
}

// ParseReadCoilsResponse parses a response to a read coils request
func (h *ProtocolHandler) ParseReadCoilsResponse(data []byte, quantity common.Quantity) ([]common.CoilValue, error) {
	// Use the parseBitResponse helper and cast the result to the expected type
	values, err := h.parseBitResponse("coils", data, quantity)
	if err != nil {
		return nil, err
	}

	// Convert []bool to []common.CoilValue (type alias, so this is a no-op in Go)
	coilValues := make([]common.CoilValue, len(values))
	for i, v := range values {
		coilValues[i] = common.CoilValue(v)
	}

	return coilValues, nil
}

// GenerateReadDiscreteInputsRequest generates a request to read discrete inputs
func (h *ProtocolHandler) GenerateReadDiscreteInputsRequest(address common.Address, quantity common.Quantity) ([]byte, error) {
	return h.generateReadRequest("discrete inputs", address, quantity, common.MaxCoilCount)
}

// ParseReadDiscreteInputsResponse parses a response to a read discrete inputs request
func (h *ProtocolHandler) ParseReadDiscreteInputsResponse(data []byte, quantity common.Quantity) ([]common.DiscreteInputValue, error) {
	// Use the parseBitResponse helper and cast the result to the expected type
	values, err := h.parseBitResponse("discrete inputs", data, quantity)
	if err != nil {
		return nil, err
	}

	// Convert []bool to []common.DiscreteInputValue (type alias, so this is a no-op in Go)
	discreteValues := make([]common.DiscreteInputValue, len(values))
	for i, v := range values {
		discreteValues[i] = common.DiscreteInputValue(v)
	}

	return discreteValues, nil
}

// GenerateReadHoldingRegistersRequest generates a request to read holding registers
func (h *ProtocolHandler) GenerateReadHoldingRegistersRequest(address common.Address, quantity common.Quantity) ([]byte, error) {
	return h.generateReadRequest("holding registers", address, quantity, common.MaxRegisterCount)
}

// ParseReadHoldingRegistersResponse parses a response to a read holding registers request
func (h *ProtocolHandler) ParseReadHoldingRegistersResponse(data []byte, quantity common.Quantity) ([]common.RegisterValue, error) {
	// Use the parseRegisterResponse helper and cast the result to the expected type
	values, err := h.parseRegisterResponse("holding registers", data, quantity)
	if err != nil {
		return nil, err
	}

	// Convert []uint16 to []common.RegisterValue (type alias, so this is a no-op in Go)
	registerValues := make([]common.RegisterValue, len(values))
	for i, v := range values {
		registerValues[i] = common.RegisterValue(v)
	}

	return registerValues, nil
}

// GenerateReadInputRegistersRequest generates a request to read input registers
func (h *ProtocolHandler) GenerateReadInputRegistersRequest(address common.Address, quantity common.Quantity) ([]byte, error) {
	return h.generateReadRequest("input registers", address, quantity, common.MaxRegisterCount)
}

// ParseReadInputRegistersResponse parses a response to a read input registers request
func (h *ProtocolHandler) ParseReadInputRegistersResponse(data []byte, quantity common.Quantity) ([]common.InputRegisterValue, error) {
	// Use the parseRegisterResponse helper and cast the result to the expected type
	values, err := h.parseRegisterResponse("input registers", data, quantity)
	if err != nil {
		return nil, err
	}

	// Convert []uint16 to []common.InputRegisterValue (type alias, so this is a no-op in Go)
	inputValues := make([]common.InputRegisterValue, len(values))
	for i, v := range values {
		inputValues[i] = common.InputRegisterValue(v)
	}

	return inputValues, nil
}

// GenerateWriteSingleCoilRequest generates a request to write a single coil
func (h *ProtocolHandler) GenerateWriteSingleCoilRequest(address common.Address, value common.CoilValue) ([]byte, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Generating write single coil request: address=%d, value=%t", address, value)

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], uint16(address))

	// Encode boolean as 0xFF00 (on) or 0x0000 (off)
	if value {
		binary.BigEndian.PutUint16(data[2:4], common.CoilOnU16)
	} else {
		binary.BigEndian.PutUint16(data[2:4], common.CoilOffU16)
	}

	h.logger.Debug(ctx, "Generated write single coil request data: %v", data)
	return data, nil
}

// ParseWriteSingleCoilResponse parses a response to a write single coil request
func (h *ProtocolHandler) ParseWriteSingleCoilResponse(data []byte) (common.Address, common.CoilValue, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Parsing write single coil response: data=%v", data)

	if len(data) != 4 {
		h.logger.Error(ctx, "Invalid response length for write single coil: expected 4, got %d", len(data))
		return 0, false, common.ErrInvalidResponseLength
	}

	address := common.Address(binary.BigEndian.Uint16(data[0:2]))
	value := binary.BigEndian.Uint16(data[2:4])

	switch value {
	case common.CoilOnU16:
		h.logger.Debug(ctx, "Parsed write single coil response: address=%d, value=true", address)
		return address, true, nil
	case common.CoilOffU16:
		h.logger.Debug(ctx, "Parsed write single coil response: address=%d, value=false", address)
		return address, false, nil
	default:
		h.logger.Error(ctx, "Invalid coil value in response: %d", value)
		return address, false, fmt.Errorf("invalid coil value: %d", value)
	}
}

// GenerateWriteSingleRegisterRequest generates a request to write a single register
func (h *ProtocolHandler) GenerateWriteSingleRegisterRequest(address common.Address, value common.RegisterValue) ([]byte, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Generating write single register request: address=%d, value=%d", address, value)

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], uint16(address))
	binary.BigEndian.PutUint16(data[2:4], value)

	h.logger.Debug(ctx, "Generated write single register request data: %v", data)
	return data, nil
}

// ParseWriteSingleRegisterResponse parses a response to a write single register request
func (h *ProtocolHandler) ParseWriteSingleRegisterResponse(data []byte) (common.Address, common.RegisterValue, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Parsing write single register response: data=%v", data)

	if len(data) != 4 {
		h.logger.Error(ctx, "Invalid response length for write single register: expected 4, got %d", len(data))
		return 0, 0, common.ErrInvalidResponseLength
	}

	address := common.Address(binary.BigEndian.Uint16(data[0:2]))
	value := common.RegisterValue(binary.BigEndian.Uint16(data[2:4]))

	h.logger.Debug(ctx, "Parsed write single register response: address=%d, value=%d", address, value)
	return address, value, nil
}

// GenerateWriteMultipleCoilsRequest generates a request to write multiple coils
func (h *ProtocolHandler) GenerateWriteMultipleCoilsRequest(address common.Address, values []common.CoilValue) ([]byte, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Generating write multiple coils request: address=%d, count=%d",
		address, len(values))

	if len(values) == 0 || len(values) > common.MaxCoilCount {
		h.logger.Error(ctx, "Invalid quantity for write multiple coils request: %d", len(values))
		return nil, common.ErrInvalidQuantity
	}

	// Calculate byte count and allocate data
	byteCount := int(math.Ceil(float64(len(values)) / 8.0))
	data := make([]byte, 5+byteCount)

	// Address
	binary.BigEndian.PutUint16(data[0:2], uint16(address))
	// Quantity
	binary.BigEndian.PutUint16(data[2:4], uint16(len(values)))
	// Byte count
	data[4] = byte(byteCount)

	// Pack coil values
	for i, value := range values {
		byteIndex := i / 8
		bitIndex := i % 8

		if value {
			data[5+byteIndex] |= (1 << uint(bitIndex))
		}
	}

	h.logger.Debug(ctx, "Generated write multiple coils request data: %v", data)
	return data, nil
}

// ParseWriteMultipleCoilsResponse parses a response to a write multiple coils request
func (h *ProtocolHandler) ParseWriteMultipleCoilsResponse(data []byte) (common.Address, common.Quantity, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Parsing write multiple coils response: data=%v", data)

	if len(data) != 4 {
		h.logger.Error(ctx, "Invalid response length for write multiple coils: expected 4, got %d", len(data))
		return 0, 0, common.ErrInvalidResponseLength
	}

	address := common.Address(binary.BigEndian.Uint16(data[0:2]))
	quantity := common.Quantity(binary.BigEndian.Uint16(data[2:4]))

	h.logger.Debug(ctx, "Parsed write multiple coils response: address=%d, quantity=%d", address, quantity)
	return address, quantity, nil
}

// GenerateWriteMultipleRegistersRequest generates a request to write multiple registers
func (h *ProtocolHandler) GenerateWriteMultipleRegistersRequest(address common.Address, values []common.RegisterValue) ([]byte, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Generating write multiple registers request: address=%d, count=%d",
		address, len(values))

	if len(values) == 0 || len(values) > common.MaxRegisterCount {
		h.logger.Error(ctx, "Invalid quantity for write multiple registers request: %d", len(values))
		return nil, common.ErrInvalidQuantity
	}

	// Calculate byte count
	byteCount := len(values) * 2

	// Allocate data
	data := make([]byte, 5+byteCount)

	// Address
	binary.BigEndian.PutUint16(data[0:2], uint16(address))
	// Quantity
	binary.BigEndian.PutUint16(data[2:4], uint16(len(values)))
	// Byte count
	data[4] = byte(byteCount)

	// Pack register values
	for i, value := range values {
		binary.BigEndian.PutUint16(data[5+i*2:5+i*2+2], value)
	}

	h.logger.Debug(ctx, "Generated write multiple registers request data: %v", data)
	return data, nil
}

// ParseWriteMultipleRegistersResponse parses a response to a write multiple registers request
func (h *ProtocolHandler) ParseWriteMultipleRegistersResponse(data []byte) (common.Address, common.Quantity, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Parsing write multiple registers response: data=%v", data)

	if len(data) != 4 {
		h.logger.Error(ctx, "Invalid response length for write multiple registers: expected 4, got %d", len(data))
		return 0, 0, common.ErrInvalidResponseLength
	}

	address := common.Address(binary.BigEndian.Uint16(data[0:2]))
	quantity := common.Quantity(binary.BigEndian.Uint16(data[2:4]))

	h.logger.Debug(ctx, "Parsed write multiple registers response: address=%d, quantity=%d", address, quantity)
	return address, quantity, nil
}

// GenerateReadWriteMultipleRegistersRequest generates a request to read and write multiple registers
func (h *ProtocolHandler) GenerateReadWriteMultipleRegistersRequest(readAddress common.Address, readQuantity common.Quantity, writeAddress common.Address, writeValues []common.RegisterValue) ([]byte, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Generating read/write multiple registers request: readAddress=%d, readQuantity=%d, writeAddress=%d, writeCount=%d",
		readAddress, readQuantity, writeAddress, len(writeValues))

	if readQuantity == 0 || readQuantity > common.MaxRegisterCount {
		h.logger.Error(ctx, "Invalid read quantity for read/write multiple registers request: %d", readQuantity)
		return nil, common.ErrInvalidQuantity
	}
	if len(writeValues) == 0 || len(writeValues) > common.MaxRegisterCount {
		h.logger.Error(ctx, "Invalid write quantity for read/write multiple registers request: %d", len(writeValues))
		return nil, common.ErrInvalidQuantity
	}

	// Calculate byte count
	byteCount := len(writeValues) * 2

	// Allocate data
	data := make([]byte, 9+byteCount)

	// Read address
	binary.BigEndian.PutUint16(data[0:2], uint16(readAddress))
	// Read quantity
	binary.BigEndian.PutUint16(data[2:4], uint16(readQuantity))
	// Write address
	binary.BigEndian.PutUint16(data[4:6], uint16(writeAddress))
	// Write quantity
	binary.BigEndian.PutUint16(data[6:8], uint16(len(writeValues)))
	// Byte count
	data[8] = byte(byteCount)

	// Pack register values
	for i, value := range writeValues {
		binary.BigEndian.PutUint16(data[9+i*2:9+i*2+2], value)
	}

	h.logger.Debug(ctx, "Generated read/write multiple registers request data: %v", data)
	return data, nil
}

// ParseReadWriteMultipleRegistersResponse parses a response to a read/write multiple registers request
func (h *ProtocolHandler) ParseReadWriteMultipleRegistersResponse(data []byte, readQuantity common.Quantity) ([]common.RegisterValue, error) {
	// Same implementation as ParseReadHoldingRegistersResponse
	return h.ParseReadHoldingRegistersResponse(data, readQuantity)
}

// GenerateReadExceptionStatusRequest generates a request to read the exception status
func (h *ProtocolHandler) GenerateReadExceptionStatusRequest() ([]byte, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Generating read exception status request")

	// No data for this request
	return []byte{}, nil
}

// ParseReadExceptionStatusResponse parses a response to a read exception status request
func (h *ProtocolHandler) ParseReadExceptionStatusResponse(data []byte) (byte, error) {
	ctx := context.Background()
	h.logger.Debug(ctx, "Parsing read exception status response: data=%v", data)

	if len(data) != 1 {
		h.logger.Error(ctx, "Invalid response length for read exception status: expected 1, got %d", len(data))
		return 0, common.ErrInvalidResponseLength
	}

	h.logger.Debug(ctx, "Parsed read exception status response: status=%d", data[0])
	return data[0], nil
}
