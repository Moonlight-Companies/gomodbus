package transport

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/Moonlight-Companies/gomodbus/common"
)

// Response implements the common.Response interface
type Response struct {
	TransactionID common.TransactionID
	ProtocolID    common.ProtocolID
	UnitID        common.UnitID
	PDU           *common.PDU
}

// NewResponse creates a new Response
func NewResponse(transactionID common.TransactionID, unitID common.UnitID, functionCode common.FunctionCode, data []byte) *Response {
	return &Response{
		TransactionID: transactionID,
		ProtocolID:    common.TCPProtocolIdentifier,
		UnitID:        unitID,
		PDU: &common.PDU{
			FunctionCode: functionCode,
			Data:         data,
		},
	}
}

// GetTransactionID returns the transaction ID
func (r *Response) GetTransactionID() common.TransactionID {
	return r.TransactionID
}

// GetUnitID returns the unit ID
func (r *Response) GetUnitID() common.UnitID {
	return r.UnitID
}

// GetPDU returns the PDU
func (r *Response) GetPDU() *common.PDU {
	return r.PDU
}

// Encode encodes a Response into bytes
func (r *Response) Encode() ([]byte, error) {
	// Calculate the length of the remaining data (Unit ID + PDU)
	length := uint16(1 + 1 + len(r.PDU.Data)) // Unit ID + Function Code + Data

	// Create a buffer to hold the encoded bytes
	buffer := bytes.Buffer{}

	// Write header
	if err := binary.Write(&buffer, binary.BigEndian, r.TransactionID); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.BigEndian, r.ProtocolID); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.BigEndian, length); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.BigEndian, r.UnitID); err != nil {
		return nil, err
	}

	// Write PDU
	if err := binary.Write(&buffer, binary.BigEndian, r.PDU.FunctionCode); err != nil {
		return nil, err
	}
	if _, err := buffer.Write(r.PDU.Data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// Decode decodes a Response from bytes
func (r *Response) Decode(data []byte) error {
	if len(data) < common.TCPHeaderLength {
		return common.ErrInvalidResponseLength
	}

	buffer := bytes.NewReader(data)

	// Read header
	if err := binary.Read(buffer, binary.BigEndian, &r.TransactionID); err != nil {
		return err
	}
	if err := binary.Read(buffer, binary.BigEndian, &r.ProtocolID); err != nil {
		return err
	}

	// Read length
	var length uint16
	if err := binary.Read(buffer, binary.BigEndian, &length); err != nil {
		return err
	}

	if err := binary.Read(buffer, binary.BigEndian, &r.UnitID); err != nil {
		return err
	}

	// Read function code
	functionCode := byte(0)
	if err := binary.Read(buffer, binary.BigEndian, &functionCode); err != nil {
		return err
	}

	// Read data
	pduDataLength := int(length) - 2 // -2 for UnitID and FunctionCode
	if pduDataLength < 0 {
		return common.ErrInvalidResponseLength
	}

	pduData := make([]byte, pduDataLength)
	if _, err := io.ReadFull(buffer, pduData); err != nil {
		return err
	}

	r.PDU = &common.PDU{
		FunctionCode: common.FunctionCode(functionCode),
		Data:         pduData,
	}

	return nil
}

// IsException checks if the response is an exception
func (r *Response) IsException() bool {
	return common.IsFunctionException(r.PDU.FunctionCode)
}

// GetException returns the exception code if the response is an exception
func (r *Response) GetException() common.ExceptionCode {
	if r.IsException() && len(r.PDU.Data) > 0 {
		return common.ExceptionCode(r.PDU.Data[0])
	}
	return 0
}

// ToError converts an exception response to an error
func (r *Response) ToError() error {
	if r.IsException() {
		return common.NewModbusError(r.PDU.FunctionCode, r.GetException())
	}
	return nil
}
