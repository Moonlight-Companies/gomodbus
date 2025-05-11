package transport

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"github.com/Moonlight-Companies/gomodbus/common"
)

// Request implements the common.Request interface
type Request struct {
	TransactionID common.TransactionID
	ProtocolID    common.ProtocolID
	UnitID        common.UnitID
	PDU           *common.PDU
	Create        time.Time
}

// NewRequest creates a new Request
func NewRequest(unitID common.UnitID, functionCode common.FunctionCode, data []byte) *Request {
	return &Request{
		ProtocolID: common.TCPProtocolIdentifier,
		UnitID:     unitID,
		PDU: &common.PDU{
			FunctionCode: functionCode,
			Data:         data,
		},
		Create: time.Now(),
	}
}

// GetTransactionID returns the transaction ID
func (r *Request) GetTransactionID() common.TransactionID {
	return r.TransactionID
}

// SetTransactionID sets the transaction ID
func (r *Request) SetTransactionID(id common.TransactionID) {
	r.TransactionID = id
}

// GetUnitID returns the unit ID
func (r *Request) GetUnitID() common.UnitID {
	return r.UnitID
}

// GetPDU returns the PDU
func (r *Request) GetPDU() *common.PDU {
	return r.PDU
}

// Encode encodes a Request into bytes
func (r *Request) Encode() ([]byte, error) {
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

// Decode decodes a Request from bytes
func (r *Request) Decode(data []byte) error {
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
	pduData := make([]byte, length-2) // -2 for UnitID and FunctionCode
	if _, err := io.ReadFull(buffer, pduData); err != nil {
		return err
	}

	r.PDU = &common.PDU{
		FunctionCode: common.FunctionCode(functionCode),
		Data:         pduData,
	}

	return nil
}

// GetLifetime returns the lifetime of the request
func (r *Request) GetLifetime() time.Duration {
	return time.Since(r.Create)
}

// Cancel is called when a transaction is cancelled
func (r *Request) Cancel(err error) {
	// Our transaction has timed out or some other error occurred
	// This method can be used for cleanup if needed
}