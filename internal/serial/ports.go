package serial

import (
	"go.bug.st/serial/enumerator"
)

// PortInfo holds details about a serial port.
type PortInfo struct {
	Name         string
	IsUSB        bool
	VID          string
	PID          string
	SerialNumber string
}

// ListPorts returns available serial ports.
func ListPorts() ([]PortInfo, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}

	var result []PortInfo
	for _, p := range ports {
		result = append(result, PortInfo{
			Name:         p.Name,
			IsUSB:        p.IsUSB,
			VID:          p.VID,
			PID:          p.PID,
			SerialNumber: p.SerialNumber,
		})
	}
	return result, nil
}
