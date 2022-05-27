package startech_kvm

import (
	"bytes"
	"github.com/tarm/serial"
	d "github.com/timgws/kvm-switch/server/drivers"
	"log"
	"strconv"
	"strings"
	"time"
)

const EnableDebugMode = false

type StartechConfig struct {
	SerialDevice string
	SerialBaud int
}

type StartechState struct {
	CurrentDevice int
}

// StartechKvm has been developed with a SV431DVIUDDM
// https://www.startech.com/en-au/server-management/sv431dviuddm
// should work for a few other similar devices, but YMMV.
type StartechKvm struct {
	d.Driver
	d.OutputSingle

	config    StartechConfig
	isRunning bool

	StartAttempted bool
	HasError       bool
	Error          error

	NumOfInputs  int
	NumOfOutputs int

	messages       chan string
	serialResponse chan string

	state      StartechState
	switching  bool
	switched   bool
	firstError bool
}

func NewInstance() *StartechKvm {
	return &StartechKvm{
		isRunning: false,
		Driver: d.Driver{
			Name: "Startech SV431DVIUDDM",
			ShortName: "kvm",
		},
		config: StartechConfig{
			SerialDevice: "/dev/tty.usbserial-141140",
			SerialBaud: 115200,
		},
		NumOfInputs: 4,
		NumOfOutputs: 1,
		firstError: true,
		state: StartechState{},
	}
}

// SupportsInitState is always false because these (and most other devices) do not support querying state.
// A boy can dream.
func (d *StartechKvm) SupportsInitState() bool {
	return false
}

// IsRunning will show if the device is being read from or not.
func (d *StartechKvm) IsRunning() bool {
	return d.isRunning
}

func (d *StartechKvm) GetShortName() string {
	return d.ShortName
}

// Start will initialize the connection...
func (d *StartechKvm) Start() bool {
	d.StartAttempted = true
	s_dev := d.config.SerialDevice
	s_baud := d.config.SerialBaud

	c := &serial.Config{Name: s_dev, Baud: s_baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		d.Error = err
		return false
	}

	go d.readPort(s)
	go d.init(s)

	go d.processResponses()

	go d.writePort(s)

	return true
}

// init the device.
func (d *StartechKvm) init(port *serial.Port) {
	// (Either) the startech is a bit dodge, or my USB->RS232 is a bit dodge.
	// let's send a fake command and wait for the error response.
	d.StartAttempted = true
	time.Sleep(time.Millisecond * 500)
	n, err := port.Write([]byte("HI!\r\n"))
	if err != nil {
		d.Error = err
		d.HasError = true
		log.Printf("Could not send %d bytes for driver.", n)
	}
	port.Flush()
}

func (d *StartechKvm) DriverName() string {
	return d.Name
}

// GetStatus does nothing here, because the method does not exist.
func (d *StartechKvm) GetStatus() {
}


// writePort manages a channel that allows us to send & receive data to this serial connection.
func (d *StartechKvm) writePort(port *serial.Port) {
	d.messages = make(chan string)
	log.Printf("WRITE PORT STARTED")

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
func (d *StartechKvm) readPort(port *serial.Port) {
	var command string
	for {
		buf := make([]byte, 60)
		n, err := port.Read(buf)
		if err != nil {
			log.Fatal(n, err)
		}


		b := bytes.Trim(buf, "\x00")
		tmpString := string(b)
		command = command + tmpString

		commands := strings.Split(command, "\r\n")

		if EnableDebugMode {
			log.Printf("BUFFER CONTENTS: %s tmpString %s", buf, tmpString)
			log.Printf("commands: %s", commands)
		}

		numCommands := len(commands)
		if numCommands > 1 {
			for i := range commands {
				// if we are at the last command, check that we have an empty command...
				if i == numCommands-1 {
					if commands[i] == "" {
						buf = []byte("")
						log.Printf("BUFFER: %s %q", buf, buf)
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

					log.Printf("[startech_kvm] Command #%d/%d: %s", i+1, len(commands), currentCommand)
				}
			}
		} else {
			if EnableDebugMode {
				log.Printf("Incomplete command from serial... expected newline")
			}
		}

		if EnableDebugMode {
			log.Printf("*-*-*-**-*-**-*-*-**-*-**-*-*-**-*-**-*-*-**-*-**-*-*-**-*-**-*-*-**-*-**-*-*-**-*-*")
			log.Printf("BUFFER CONTENTS: %s tmpString %s", buf, tmpString)
			log.Printf("commands: %s", commands)
			log.Printf("%q", buf)
			log.Printf("---------------------------")
		}

		//log.Printf("%s %q", buf, buf)
	}
}

// processResponses reads serial commands that have been fully read from the serial connection.
func (d *StartechKvm) processResponses() {
	d.serialResponse = make(chan string)

	go func() {
		for {
			select {
			case msg := <-d.serialResponse:
				if EnableDebugMode {
					log.Printf("<== [STARTECH] READ SERIAL COMMAND: %s %q", msg, msg)
				}

				if msg == "ERROR" {
					if !d.firstError {
						d.HasError = true
					} else {
						log.Printf("Ignore the first error, we are just initializing our state - looks like this device is correct")
						d.firstError = false
						d.isRunning = true
					}
					continue
				}

				if strings.Contains(msg, "F/W Version") {
					// unlike Blustream, we only get to know the device when it boots.
					// SV431DVIUDDM F/W Version :H2K B4.1
					version := strings.Split(msg, " ")
					if len(version) == 5 {
						fwVersion := strings.Replace(strings.Join(version[3:], " "), ":", "", 1)
						d.Driver.Name = "Startech.com " + version[0] + fwVersion
						log.Println("[startech_kvm]: New driver name is: " + d.Driver.Name)
					}
				}

				if len(msg) == 3 {
					if msg[:2] == "CH" {
						log.Println(msg, msg[2:])
						chn, err := strconv.Atoi(msg[2:])
						if err != nil {
							d.HasError = true
							d.Error = err
						}

						d.state.CurrentDevice = chn
					}
				}
			}
		}
	}()
}

func (d *StartechKvm) SetOutput(inputName string) {
	d.messages <- "CH" + inputName
}

func (d *StartechKvm) LastError() error {
	return d.Error
}

func quickLog(str string) {
	log.Println(str)
}