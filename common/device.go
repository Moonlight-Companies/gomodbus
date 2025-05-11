package common

// DeviceIDObject represents a single device identification object
type DeviceIDObject struct {
	ID     DeviceIDObjectCode // Object ID
	Length byte               // Length of the object data
	Value  string             // Object value (string)
}

// DeviceIdentification represents a complete device identification response
type DeviceIdentification struct {
	ReadDeviceIDCode   ReadDeviceIDCode  // The read device ID code used in the request
	ConformityLevel    byte              // Device conformity level
	MoreFollows        bool              // Indicates if more data follows (for stream access)
	NextObjectID       DeviceIDObjectCode // The next object ID (for stream access when MoreFollows is true)
	NumberOfObjects    byte              // Number of objects in this response
	Objects            []DeviceIDObject  // The list of device identification objects
}

// GetObject returns the object with the specified ID, or nil if not found
func (d *DeviceIdentification) GetObject(id DeviceIDObjectCode) *DeviceIDObject {
	for i := range d.Objects {
		if d.Objects[i].ID == id {
			return &d.Objects[i]
		}
	}
	return nil
}

// GetVendorName returns the vendor name (object ID 0x00), or empty string if not present
func (d *DeviceIdentification) GetVendorName() string {
	obj := d.GetObject(DeviceIDVendorName)
	if obj != nil {
		return obj.Value
	}
	return ""
}

// GetProductCode returns the product code (object ID 0x01), or empty string if not present
func (d *DeviceIdentification) GetProductCode() string {
	obj := d.GetObject(DeviceIDProductCode)
	if obj != nil {
		return obj.Value
	}
	return ""
}

// GetRevision returns the major/minor revision (object ID 0x02), or empty string if not present
func (d *DeviceIdentification) GetRevision() string {
	obj := d.GetObject(DeviceIDMajorMinorRevision)
	if obj != nil {
		return obj.Value
	}
	return ""
}

// GetVendorURL returns the vendor URL (object ID 0x03), or empty string if not present
func (d *DeviceIdentification) GetVendorURL() string {
	obj := d.GetObject(DeviceIDVendorURL)
	if obj != nil {
		return obj.Value
	}
	return ""
}

// GetProductName returns the product name (object ID 0x04), or empty string if not present
func (d *DeviceIdentification) GetProductName() string {
	obj := d.GetObject(DeviceIDProductName)
	if obj != nil {
		return obj.Value
	}
	return ""
}

// GetModelName returns the model name (object ID 0x05), or empty string if not present
func (d *DeviceIdentification) GetModelName() string {
	obj := d.GetObject(DeviceIDModelName)
	if obj != nil {
		return obj.Value
	}
	return ""
}

// GetUserApplicationName returns the user application name (object ID 0x06), or empty string if not present
func (d *DeviceIdentification) GetUserApplicationName() string {
	obj := d.GetObject(DeviceIDUserAppName)
	if obj != nil {
		return obj.Value
	}
	return ""
}