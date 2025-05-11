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

// HandleReadDeviceIdentification processes a read device identification request
func (h *serverProtocolHandler) HandleReadDeviceIdentification(ctx context.Context, req common.Request, store common.DataStore) (common.Response, error) {
	// Parse request data
	if len(req.GetPDU().Data) < 3 {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	// Data format:
	// Byte 0: MEI Type (0x0E for ReadDeviceID)
	// Byte 1: ReadDeviceID code (0x01-0x04)
	// Byte 2: Object ID
	if common.MEIType(req.GetPDU().Data[0]) != common.MEIReadDeviceID {
		return nil, common.NewModbusError(req.GetPDU().FunctionCode, common.ExceptionInvalidDataValue)
	}

	readDeviceIDCode := common.ReadDeviceIDCode(req.GetPDU().Data[1])
	objectID := common.DeviceIDObjectCode(req.GetPDU().Data[2])

	// Create a response based on the request
	// Fixed values for this example server
	deviceID := &common.DeviceIdentification{
		ReadDeviceIDCode: readDeviceIDCode,
		ConformityLevel:  0x01, // Basic identification
		MoreFollows:      false,
		NextObjectID:     0x00,
		NumberOfObjects:  0,
		Objects:          make([]common.DeviceIDObject, 0),
	}

	// Default objects to include (all basic identification objects)
	objectsToInclude := []common.DeviceIDObjectCode{
		common.DeviceIDVendorName,
		common.DeviceIDProductCode,
		common.DeviceIDMajorMinorRevision,
	}

	// Specific object handling
	if readDeviceIDCode == common.ReadDeviceIDSpecificObject {
		objectsToInclude = []common.DeviceIDObjectCode{objectID}
	} else if readDeviceIDCode == common.ReadDeviceIDRegularStream {
		// Include regular objects too
		objectsToInclude = append(objectsToInclude,
			common.DeviceIDVendorURL,
			common.DeviceIDProductName,
			common.DeviceIDModelName,
			common.DeviceIDUserAppName,
		)
	} else if readDeviceIDCode == common.ReadDeviceIDExtendedStream {
		// Include all objects (basic + regular + extended)
		objectsToInclude = append(objectsToInclude,
			common.DeviceIDVendorURL,
			common.DeviceIDProductName,
			common.DeviceIDModelName,
			common.DeviceIDUserAppName,
			// Add any extended objects here (0x80-0xFF)
			common.DeviceIDObjectCode(0x80), // Example extended object
		)
	}

	// Fixed object values for this server
	objectValues := map[common.DeviceIDObjectCode]string{
		// Basic identification objects (mandatory)
		common.DeviceIDVendorName:         "GoModbus",
		common.DeviceIDProductCode:        "GM-001",
		common.DeviceIDMajorMinorRevision: "1.0",

		// Regular identification objects (optional)
		common.DeviceIDVendorURL:          "https://github.com/Moonlight-Companies/gomodbus",
		common.DeviceIDProductName:        "GoModbus Server",
		common.DeviceIDModelName:          "Modbus TCP Server",
		common.DeviceIDUserAppName:        "Example Server",

		// Extended identification objects (vendor-specific)
		common.DeviceIDObjectCode(0x80):   "Extended Object Example",
	}

	// Add objects to response
	for _, id := range objectsToInclude {
		value, exists := objectValues[id]
		if exists {
			deviceID.Objects = append(deviceID.Objects, common.DeviceIDObject{
				ID:     id,
				Length: byte(len(value)),
				Value:  value,
			})
		}
	}

	deviceID.NumberOfObjects = byte(len(deviceID.Objects))

	// Encode response
	// Format:
	// Byte 0: MEI Type (0x0E)
	// Byte 1: ReadDeviceID code
	// Byte 2: Conformity level
	// Byte 3: More follows (0 or 1)
	// Byte 4: Next object ID
	// Byte 5: Number of objects
	// For each object:
	//   Byte n+0: Object ID
	//   Byte n+1: Object length
	//   Byte n+2..n+1+length: Object value

	// Calculate response size
	responseSize := 6 // Fixed header
	for _, obj := range deviceID.Objects {
		responseSize += 2 + int(obj.Length) // ID + length + value
	}

	responseData := make([]byte, responseSize)
	responseData[0] = byte(common.MEIReadDeviceID)
	responseData[1] = byte(deviceID.ReadDeviceIDCode)
	responseData[2] = deviceID.ConformityLevel
	if deviceID.MoreFollows {
		responseData[3] = 1
	} else {
		responseData[3] = 0
	}
	responseData[4] = byte(deviceID.NextObjectID)
	responseData[5] = deviceID.NumberOfObjects

	// Add objects
	offset := 6
	for _, obj := range deviceID.Objects {
		responseData[offset] = byte(obj.ID)
		responseData[offset+1] = obj.Length
		copy(responseData[offset+2:offset+2+int(obj.Length)], []byte(obj.Value))
		offset += 2 + int(obj.Length)
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
