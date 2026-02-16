package serial

import (
	"io"
	"sync"

	"go.bug.st/serial"
)

// DataReceivedMsg is sent when data arrives from the serial port.
type DataReceivedMsg struct {
	Data string
}

// Monitor manages a serial port connection.
type Monitor struct {
	port     serial.Port
	portName string
	baudRate int
	mu       sync.Mutex
	running  bool
	dataCh   chan string
	done     chan struct{}
}

// NewMonitor creates a new serial monitor.
func NewMonitor() *Monitor {
	return &Monitor{
		dataCh: make(chan string, 64),
		done:   make(chan struct{}),
	}
}

// Connect opens a serial port with the given settings.
func (m *Monitor) Connect(portName string, baudRate int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.disconnectLocked()
	}

	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return err
	}

	m.port = port
	m.portName = portName
	m.baudRate = baudRate
	m.running = true
	m.done = make(chan struct{})

	go m.readLoop()
	return nil
}

// Disconnect closes the serial port.
func (m *Monitor) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectLocked()
}

func (m *Monitor) disconnectLocked() {
	if !m.running {
		return
	}
	m.running = false
	if m.port != nil {
		m.port.Close()
	}
	close(m.done)
}

// Write sends data to the serial port.
func (m *Monitor) Write(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.port == nil {
		return io.ErrClosedPipe
	}
	_, err := m.port.Write(data)
	return err
}

// DataChan returns the channel that receives serial data.
func (m *Monitor) DataChan() <-chan string {
	return m.dataCh
}

// Connected returns whether the monitor is connected.
func (m *Monitor) Connected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *Monitor) readLoop() {
	buf := make([]byte, 1024)
	for {
		select {
		case <-m.done:
			return
		default:
		}

		n, err := m.port.Read(buf)
		if err != nil {
			return
		}
		if n > 0 {
			select {
			case m.dataCh <- string(buf[:n]):
			default:
				// Drop data if channel is full
			}
		}
	}
}
