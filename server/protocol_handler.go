package server

import (
	"context"
	"encoding/binary"
	"math"

	"github.com/Moonlight-Companies/gomodbus/common"
	"github.com/Moonlight-Companies/gomodbus/transport"
)

// serverProtocolHandler processes Modbus requests and generates responses
type serverProtocolHandler struct{}

// newServerProtocolHandler creates a new protocol handler for server
func newServerProtocolHandler() *serverProtocolHandler {
	return &serverProtocolHandler{}
}

// handleReadBitValues is a helper function for handling bit value read requests (coils, discrete inputs)
func (h *serverProtocolHandler) handleReadBitValues(
	ctx context.Context,
	req common.Request,
	store common.DataStore,
	itemType string,
	maxQuantity common.Quantity,
	readFunc func(context.Context, common.Address, common.Quantity) ([]bool, error)) (common.Response, error) {

	// Parse request data
	if len(req.GetPDU().Data) != 4 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	address := common.Address(binary.BigEndian.Uint16(req.GetPDU().Data[0:2]))
	quantity := common.Quantity(binary.BigEndian.Uint16(req.GetPDU().Data[2:4]))

	if quantity == 0 || quantity > maxQuantity {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Read values from data store
	values, err := readFunc(ctx, address, quantity)
	if err != nil {
		if err == common.ErrInvalidQuantity {
			return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
		}
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionServerDeviceFailure)
	}

	// Calculate response data size and create response data
	byteCount := int(math.Ceil(float64(quantity) / 8.0))
	responseData := make([]byte, 1+byteCount)
	responseData[0] = byte(byteCount) // First byte is the byte count

	// Pack bit values into bytes
	for i, value := range values {
		if value {
			byteIndex := i / 8
			bitOffset := i % 8
			responseData[1+byteIndex] |= (1 << uint(bitOffset))
		}
	}

	// Create the response
	response := transport.NewResponse(
		req.GetTransactionID(),
		req.GetUnitID(),
		req.GetPDU().FunctionCode,
		responseData,
	)

	return response, nil
}

// handleReadRegisterValues is a helper function for handling register read requests (holding/input registers)
func (h *serverProtocolHandler) handleReadRegisterValues(
	ctx context.Context,
	req common.Request,
	store common.DataStore,
	itemType string,
	maxQuantity common.Quantity,
	readFunc func(context.Context, common.Address, common.Quantity) ([]uint16, error)) (common.Response, error) {

	// Parse request data
	if len(req.GetPDU().Data) != 4 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	address := common.Address(binary.BigEndian.Uint16(req.GetPDU().Data[0:2]))
	quantity := common.Quantity(binary.BigEndian.Uint16(req.GetPDU().Data[2:4]))

	if quantity == 0 || quantity > maxQuantity {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Read registers from data store
	values, err := readFunc(ctx, address, quantity)
	if err != nil {
		if err == common.ErrInvalidQuantity {
			return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
		}
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionServerDeviceFailure)
	}

	// Calculate response data size and create response data
	byteCount := len(values) * 2
	responseData := make([]byte, 1+byteCount)
	responseData[0] = byte(byteCount) // First byte is the byte count

	// Pack register values into bytes
	for i, value := range values {
		binary.BigEndian.PutUint16(responseData[1+i*2:1+i*2+2], value)
	}

	// Create the response
	response := transport.NewResponse(
		req.GetTransactionID(),
		req.GetUnitID(),
		req.GetPDU().FunctionCode,
		responseData,
	)

	return response, nil
}

// HandleReadCoils processes a read coils request
func (h *serverProtocolHandler) HandleReadCoils(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	return h.handleReadBitValues(
		ctx,
		req,
		store,
		"coils",
		common.MaxCoilCount,
		store.ReadCoils,
	)
}

// HandleReadDiscreteInputs processes a read discrete inputs request
func (h *serverProtocolHandler) HandleReadDiscreteInputs(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	return h.handleReadBitValues(
		ctx,
		req,
		store,
		"discrete inputs",
		common.MaxCoilCount,
		store.ReadDiscreteInputs,
	)
}

// HandleReadHoldingRegisters processes a read holding registers request
func (h *serverProtocolHandler) HandleReadHoldingRegisters(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	return h.handleReadRegisterValues(
		ctx,
		req,
		store,
		"holding registers",
		common.MaxRegisterCount,
		store.ReadHoldingRegisters,
	)
}

// HandleReadInputRegisters processes a read input registers request
func (h *serverProtocolHandler) HandleReadInputRegisters(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	return h.handleReadRegisterValues(
		ctx,
		req,
		store,
		"input registers",
		common.MaxRegisterCount,
		store.ReadInputRegisters,
	)
}

// HandleWriteSingleCoil processes a write single coil request
func (h *serverProtocolHandler) HandleWriteSingleCoil(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	// Parse request data
	if len(req.GetPDU().Data) != 4 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	address := common.Address(binary.BigEndian.Uint16(req.GetPDU().Data[0:2]))
	value := binary.BigEndian.Uint16(req.GetPDU().Data[2:4])

	// Check that value is either 0x0000 (OFF) or 0xFF00 (ON)
	var coilValue common.CoilValue
	if value == common.CoilOnU16 {
		coilValue = true
	} else if value == common.CoilOffU16 {
		coilValue = false
	} else {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Write the coil value to the data store
	err := store.WriteSingleCoil(ctx, address, coilValue)
	if err != nil {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionServerDeviceFailure)
	}

	// Create the response (echo the request)
	response := transport.NewResponse(
		req.GetTransactionID(),
		req.GetUnitID(),
		req.GetPDU().FunctionCode,
		req.GetPDU().Data,
	)

	return response, nil
}

// HandleWriteSingleRegister processes a write single register request
func (h *serverProtocolHandler) HandleWriteSingleRegister(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	// Parse request data
	if len(req.GetPDU().Data) != 4 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	address := common.Address(binary.BigEndian.Uint16(req.GetPDU().Data[0:2]))
	value := common.RegisterValue(binary.BigEndian.Uint16(req.GetPDU().Data[2:4]))

	// Write the register value to the data store
	err := store.WriteSingleRegister(ctx, address, value)
	if err != nil {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionServerDeviceFailure)
	}

	// Create the response (echo the request)
	response := transport.NewResponse(
		req.GetTransactionID(),
		req.GetUnitID(),
		req.GetPDU().FunctionCode,
		req.GetPDU().Data,
	)

	return response, nil
}

// HandleWriteMultipleCoils processes a write multiple coils request
func (h *serverProtocolHandler) HandleWriteMultipleCoils(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	// Parse request data
	if len(req.GetPDU().Data) < 5 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	address := common.Address(binary.BigEndian.Uint16(req.GetPDU().Data[0:2]))
	quantity := common.Quantity(binary.BigEndian.Uint16(req.GetPDU().Data[2:4]))
	byteCount := int(req.GetPDU().Data[4])

	// Validate data length
	if len(req.GetPDU().Data) != 5+byteCount {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Check quantity limits
	if quantity == 0 || quantity > common.MaxCoilCount {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Validate byte count matches quantity
	expectedByteCount := int(math.Ceil(float64(quantity) / 8.0))
	if byteCount != expectedByteCount {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Extract coil values from request
	values := make([]common.CoilValue, quantity)
	for i := uint16(0); i < uint16(quantity); i++ {
		byteIndex := i / 8
		bitOffset := i % 8
		values[i] = (req.GetPDU().Data[5+byteIndex]>>uint(bitOffset))&0x01 != 0
	}

	// Write the coil values to the data store
	err := store.WriteMultipleCoils(ctx, address, values)
	if err != nil {
		if err == common.ErrInvalidQuantity {
			return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
		}
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionServerDeviceFailure)
	}

	// Create the response
	responseData := make([]byte, 4)
	binary.BigEndian.PutUint16(responseData[0:2], uint16(address))
	binary.BigEndian.PutUint16(responseData[2:4], uint16(quantity))

	response := transport.NewResponse(
		req.GetTransactionID(),
		req.GetUnitID(),
		req.GetPDU().FunctionCode,
		responseData,
	)

	return response, nil
}

// HandleWriteMultipleRegisters processes a write multiple registers request
func (h *serverProtocolHandler) HandleWriteMultipleRegisters(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	// Parse request data
	if len(req.GetPDU().Data) < 5 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	address := common.Address(binary.BigEndian.Uint16(req.GetPDU().Data[0:2]))
	quantity := common.Quantity(binary.BigEndian.Uint16(req.GetPDU().Data[2:4]))
	byteCount := int(req.GetPDU().Data[4])

	// Validate data length
	if len(req.GetPDU().Data) != 5+byteCount {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Check quantity limits
	if quantity == 0 || quantity > common.MaxRegisterCount {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Validate byte count matches quantity
	if byteCount != int(quantity)*2 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Extract register values from request
	values := make([]common.RegisterValue, quantity)
	for i := uint16(0); i < uint16(quantity); i++ {
		values[i] = common.RegisterValue(binary.BigEndian.Uint16(req.GetPDU().Data[5+i*2 : 5+i*2+2]))
	}

	// Write the register values to the data store
	err := store.WriteMultipleRegisters(ctx, address, values)
	if err != nil {
		if err == common.ErrInvalidQuantity {
			return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
		}
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionServerDeviceFailure)
	}

	// Create the response
	responseData := make([]byte, 4)
	binary.BigEndian.PutUint16(responseData[0:2], uint16(address))
	binary.BigEndian.PutUint16(responseData[2:4], uint16(quantity))

	response := transport.NewResponse(
		req.GetTransactionID(),
		req.GetUnitID(),
		req.GetPDU().FunctionCode,
		responseData,
	)

	return response, nil
}

// HandleReadWriteMultipleRegisters processes a read/write multiple registers request
func (h *serverProtocolHandler) HandleReadWriteMultipleRegisters(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	// Parse request data
	if len(req.GetPDU().Data) < 9 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	readAddress := common.Address(binary.BigEndian.Uint16(req.GetPDU().Data[0:2]))
	readQuantity := common.Quantity(binary.BigEndian.Uint16(req.GetPDU().Data[2:4]))
	writeAddress := common.Address(binary.BigEndian.Uint16(req.GetPDU().Data[4:6]))
	writeQuantity := common.Quantity(binary.BigEndian.Uint16(req.GetPDU().Data[6:8]))
	byteCount := int(req.GetPDU().Data[8])

	// Validate data length
	if len(req.GetPDU().Data) != 9+byteCount {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Check quantity limits
	if readQuantity == 0 || readQuantity > common.MaxRegisterCount ||
		writeQuantity == 0 || writeQuantity > common.MaxRegisterCount {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Validate byte count matches write quantity
	if byteCount != int(writeQuantity)*2 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Extract register values from request
	writeValues := make([]common.RegisterValue, writeQuantity)
	for i := uint16(0); i < uint16(writeQuantity); i++ {
		writeValues[i] = common.RegisterValue(binary.BigEndian.Uint16(req.GetPDU().Data[9+i*2 : 9+i*2+2]))
	}

	// Write the register values to the data store
	err := store.WriteMultipleRegisters(ctx, writeAddress, writeValues)
	if err != nil {
		if err == common.ErrInvalidQuantity {
			return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
		}
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionServerDeviceFailure)
	}

	// Read the register values from the data store
	readValues, err := store.ReadHoldingRegisters(ctx, readAddress, readQuantity)
	if err != nil {
		if err == common.ErrInvalidQuantity {
			return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
		}
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionServerDeviceFailure)
	}

	// Calculate response data size and create response data
	byteCount = len(readValues) * 2
	responseData := make([]byte, 1+byteCount)
	responseData[0] = byte(byteCount) // First byte is the byte count

	// Pack register values into bytes
	for i, value := range readValues {
		binary.BigEndian.PutUint16(responseData[1+i*2:1+i*2+2], value)
	}

	// Create the response
	response := transport.NewResponse(
		req.GetTransactionID(),
		req.GetUnitID(),
		req.GetPDU().FunctionCode,
		responseData,
	)

	return response, nil
}
