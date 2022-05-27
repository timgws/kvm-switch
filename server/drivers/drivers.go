package drivers

type Driver struct {
	DriverInterface
	Name string
	ShortName string

	Inputs  []Input
	Outputs []Output
}

type Input struct {
	DriverInput
	InputName string
	Active bool
}

type Output struct {
	DriverOutput
	OutputName string
	Active bool
	Input DriverInput
}

type DriverInput interface {

}

type DriverOutput interface {

}

type DriverInterface interface {
	DriverName() string
	GetShortName() string

	// IsRunning will query if a driver is currently operational.
	// If a driver is not running, it might mean that the device that it was connected to is gone (eg, USB Serial device)
	IsRunning() 		bool

	// SupportsInitState determines if a driver supports telling us about the state of the controlling device.
	// If we support knowing the state, we will query it and update our own state about the device.
	// If the device/driver does not implement querying state, we will simply reset all devices to the initial state.
	SupportsInitState() bool

	// Start running a driver. Attempt to Dial the device, and determine the state (if possible)
	Start() 			bool

	// Shutdown disconnects from a controlling device.
	Shutdown()  		bool

	GetStatus()

	LastError()			error

	IsMatrix() bool
}

type OutputMatrix interface {
	SetOutput(outputName string, inputName string)
}

type OutputSingle interface {
	SetOutput(inputName string)
}