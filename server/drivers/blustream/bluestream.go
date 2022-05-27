package blustream

import (
	"bytes"
	"github.com/tarm/serial"
	d "github.com/timgws/kvm-switch/server/drivers"
	"log"
	"regexp"
	"strings"
	"time"
)

const EnableDebugMode = false

// Reading keeps track of where the statemachine is in reading the `status` command from the BlueStream.
type Reading int
const (
	ReadingModel Reading = iota
	ReadingVersion
	WaitingInput
	ReadingInput
	ReadingOutput
)

// BlustreamConfig holds the device-specific configuration.
type BlustreamConfig struct {
	SerialDevice string
	SerialBaud int
}

// BlustreamInput represents a HDMI/DVI/USB-C input on a given Blustream matrix
type BlustreamInput struct {
	*d.Input
	Edid string
}

// BlustreamOutput represents a HDMI/DVI/USB-C input on a given Blustream matrix
type BlustreamOutput struct {
	*d.Output
	Edid string
}

// BlustreamState holds state about the current instance of a Blustream device.
type BlustreamState struct {
	CurrentDevice int

	Model string
	ModelName string
	CurrentVersion string

	readingInputNumber int
	readingOutputNumber int
}

// BlustreamMatrix has been developed with a cmx44ab
// https://www.blustream.com.au/cmx44ab
// should work for a few other similar devices, but YMMV.
type BlustreamMatrix struct {
	*d.Driver
	d.OutputMatrix

	config BlustreamConfig
	// isRunning defines whether we are connected to the rs232 port from the device, and it is working as expected.
	isRunning bool

	// StartAttempted defines if we have attempted to start connecting to the serial port, but the device has not init.
	StartAttempted bool

	// HasError is true if the device and/or connection has an error.
	HasError bool

	// Error contains the error of the device
	// WARNING: Error may not contain text! Check HasError.
	Error error

	// NumOfInputs: The number of inputs present on the device
	NumOfInputs int
	// NumOfOutputs: The number of outputs present on the device
	NumOfOutputs int

	// messages are commands that need to be sent down the channel to the serial device.
	messages chan string

	// serialResponse contains text that is coming inbound from the Blustream device.
	serialResponse chan string

	// finishedSwap makes sure that matrix swaps are synchronous
	finishedSwap chan bool

	state           BlustreamState
	switching       bool
	switched        bool
	statusIncoming  bool
	statusConfirmed bool
	statusStarted   bool
	statusReading   Reading
	modelSet        bool

	// port contains the RS232 connection
	port *serial.Port

	// updatedInputs show inputs that have recently been updated (eg, new HDMI device coming online).
	updatedInputs []int

	// Inputs are a list of all the Driver.Inputs that are present on the matrix.
	Inputs  []BlustreamInput

	// Outputs are a list of all the Driver.Outputs that are present on the matrix.
	Outputs []BlustreamOutput
}

// NewInstance create a new instance of a Blustream device.
func NewInstance() *BlustreamMatrix {
	return &BlustreamMatrix{
		isRunning: false,
		Driver: &d.Driver{
			Name: "Blustream",
			ShortName: "matrix",
		},
		config: BlustreamConfig{
			SerialDevice: "/dev/tty.usbserial-141130",
			SerialBaud: 57600,
		},
		state: BlustreamState{},
	}
}

// SupportsInitState is always false because these (and most other devices) do not support querying state.
// A boy can dream.
func (d *BlustreamMatrix) SupportsInitState() bool {
	return true
}

func (d *BlustreamMatrix) GetShortName() string {
	return d.ShortName
}

func (d *BlustreamMatrix) DriverName() string {
	return d.Name
}

// IsRunning will show if the device is being read from or not.
func (d *BlustreamMatrix) IsRunning() bool {
	return d.isRunning
}

// Start initializes the connection, sends first status command.
func (d *BlustreamMatrix) Start() bool {
	d.StartAttempted = true
	s_dev := d.config.SerialDevice
	s_baud := d.config.SerialBaud

	c := &serial.Config{Name: s_dev, Baud: s_baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		d.Error = err
		return false
	}

	d.port = s

	go d.readPort()
	go d.init()

	//go d.pingStatus(s)

	go d.processResponses()

	go d.writePort(s)

	return true
}

// GetStatus ask the Blustream matrix what the current state of the device is.
// Call me to see if devices have changes (without notifying the switch)
func (d *BlustreamMatrix) GetStatus() {
	d.pingStatus()
}

// SetOutput will change the output of a port to the given input port.
func (d *BlustreamMatrix) SetOutput(outputName string, inputName string) {
	if d.switching {
		return
	}
	debugLog("OUTPUT MATRIX %s -> %s", outputName, inputName)

	d.finishedSwap = make(chan bool)

	var matrixOutput *BlustreamOutput
	var matrixInput *BlustreamInput

	for i, output := range d.Outputs {
		if output.OutputName == outputName {
			debugLog("<- OUTPUT %s == %s", output.OutputName, outputName)
			matrixOutput = &d.Outputs[i]
		}
	}
	for i, input := range d.Inputs {
		if input.InputName == inputName {
			debugLog("<- INPUT %s == %s", input.InputName, inputName)
			matrixInput = &d.Inputs[i]
		}
	}

	if matrixInput != nil && matrixOutput != nil {
		if !d.switching {
			d.switching = true
			d.messages <- "OUT" + outputName + "FR" + inputName
		}
	}

	select {
	case _, ok := <-d.finishedSwap:
		if ok {
			return
		}
	}
}

// init the device.
func (d *BlustreamMatrix) init() {
	port := d.port
	// We are going to ask the device for the current status.
	time.Sleep(time.Millisecond * 500)
	d.statusIncoming = true
	d.statusReading = ReadingModel
	n, err := port.Write([]byte("STATUS\r\n"))
	if err != nil {
		d.statusReading = WaitingInput
		d.Error = err
		d.HasError = true
		log.Printf("Could not send %d bytes for driver.", n)
	}
	port.Flush()
}

// init the device.
func (d *BlustreamMatrix) pingStatus() {
	port := d.port
	// We are going to ask the device for the current status.
	time.Sleep(time.Millisecond * 500)
	d.statusIncoming = true
	n, err := port.Write([]byte("STATUS\r\n"))
	if err != nil {
		d.statusReading = WaitingInput
		d.Error = err
		d.HasError = true
		log.Printf("Could not send %d bytes for driver.", n)
	}
	port.Flush()
}


// writePort manages a channel that allows us to send & receive data to this serial connection.
func (d *BlustreamMatrix) writePort(port *serial.Port) {
	d.messages = make(chan string)

	go func() {
		for {
			select {
			case msg := <-d.messages:
				log.Printf("==> WRITE PORT MSG: %s", msg)
				n, err := port.Write([]byte(msg + "\r\n"))
				if err != nil {
					log.Printf("Error writing %d bytes: %s", n, err)
					d.HasError = true
					d.Error = err
				}
			}
		}
	}()
}

// readPort continuously reads data from the serial port, checks if there is some new data received
// and sends it down to the serialResponse channel when we are g2g with a response from a command.
//
// TODO: This needs a big refactor. It's a little dodgy, but it does the trick.
func (d *BlustreamMatrix) readPort() {
	port := *d.port
	var command string
	for {
		buf := make([]byte, 820)
		n, err := port.Read(buf)
		if err != nil {
			log.Fatal(n, err)
		}


		b := bytes.Trim(buf, "\x00")
		tmpString := string(b)
		command = command + tmpString

		commands := strings.Split(command, "\r\n")

		debugLog("BUFFER CONTENTS: %s tmpString %s", buf, tmpString)
		debugLog("commands: %s", commands)

		numCommands := len(commands)
		if numCommands > 1 {
			for i := range commands {
				// if we are at the last command, check that we have an empty command...
				if i == numCommands-1 {
					if commands[i] == "" {
						buf = []byte("")
						debugLog("BUFFER: %s %q", buf, buf)
						continue
					}
				} else if i != numCommands {
					currentCommand := commands[0]
					if len(commands) > 1 {
						numCommands--
						commands = commands[1:]

						//buf = []byte(string(commands.joi))
						command = strings.Join(commands, "\r\n")
						buf = []byte(command)
					}

					d.serialResponse <- currentCommand

					log.Printf("[blustream] Command #%d/%d: %s", i+1, len(commands), currentCommand)
				}
			}
		} else {
			debugLog("Incomplete command from serial... expected newline")
		}

		if EnableDebugMode {
			log.Printf("🧐 Driver is in debug mode. Here is our current buffer/commands contents")
			log.Printf("BUFFER CONTENTS: %s tmpString: %s", buf, tmpString)
			log.Printf("commands: %s", commands)
			log.Printf("%q", buf)
			log.Printf("---------------------------")
		}

		//log.Printf("%s %q", buf, buf)
	}
}

// processResponses reads serial commands that have been fully read from the serial connection.
func (d *BlustreamMatrix) processResponses() {
	d.serialResponse = make(chan string)

	go func() {
		for {
			select {
			case msg := <-d.serialResponse:
				debugLog("<== READ SERIAL COMMAND: %s %q", msg, msg)

				// Sometimes the output contains the model name. Hope we have it!
				if strings.Contains(msg, "> ") && len(d.state.ModelName) > 0 {
					s := strings.SplitN(msg, "> ", 2)
					if s[0] == d.state.ModelName {
						msg = strings.Join(s[1:], "> ")
					}
				}

				// NOTE status-response.txt to see what we are parsing.
				if d.statusIncoming && msg == "STATUS" {
					d.statusConfirmed = true
					d.statusStarted = false
					d.statusReading = ReadingModel
				} else if d.statusConfirmed && len(msg) > 2 {
					if msg[:2] == "==" {
						// for the first set of "==", we need to stay in the status incoming state.
						// for the second, we can leave this special state.
						if !d.statusStarted {
							d.statusIncoming = false
							d.statusConfirmed = false
						}
						d.statusStarted = true
						d.statusConfirmed = true
						continue
					}

					d.readStatus(msg)
					continue
				}

				if len(msg) > 10 {
					// [SUCCESS]Set output 01 connect from input 02.
					if msg[:9] == "[SUCCESS]" {
						//match := []byte(msg[9:])
						r, _ := regexp.Compile(`Set output (\d*) connect from input (\d*)`)
						f := r.FindAllStringSubmatch(msg[9:], 12)
						if len(f[0]) == 2 {
							log.Printf("Swapped input %s to output %s", f[0][2], f[0][1])
						}
						d.finishedSwap <- true
						d.switching = false
						d.switched = true
					}
				}
			}
		}
	}()
}

// LastError return the last
func (d *BlustreamMatrix) LastError() error {
	return d.Error
}

// GetInput will return an input with a given name
func (d *BlustreamMatrix) GetInput(inputName string) *BlustreamInput {
	for _, input := range d.Inputs {
		if input.InputName == inputName {
			return &input
		}
	}
	return nil
}

// debugLog will output something only if EnableDebugMode is true.
func debugLog(msg string, v ...interface{}) {
	if EnableDebugMode {
		log.Printf(msg, v)
	}
}