package common

// PDU (Protocol Data Unit) is the core Modbus message structure
type PDU struct {
	FunctionCode FunctionCode
	Data         []byte
}
