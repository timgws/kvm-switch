package blustream

import (
	"github.com/timgws/kvm-switch/server/drivers"
	"log"
	"strconv"
	"strings"
)

// readStatus will read each line from the serial console after the `status` command is issued.
func (d *BlustreamMatrix) readStatus(_msg string) {
	msg := strings.TrimSpace(_msg)

	if !d.statusIncoming {
		d.NumOfOutputs = len(d.Outputs)
	}

	// If our line contains the 'Status' word, it also contains the model name.
	// Replace the name in the driver.
	if d.statusReading == ReadingModel && strings.Contains(msg, "Status") {
		d.setModelState(msg)
	}

	if d.statusReading == ReadingVersion && strings.Contains(msg, "Version") {
		d.setFirmwareState(msg)
		d.statusReading = WaitingInput
		return
	}

	// Input   Edid         HDMIcon
	if d.statusReading == WaitingInput && strings.Contains(msg, "Input") {
		res := strings.Fields(msg)
		if strings.ToTitle(res[0]) == "INPUT" {
			d.statusReading = ReadingInput
			d.state.readingInputNumber = 0
			return
		}
	}

	if d.statusReading == ReadingInput {
		res := strings.Fields(msg)

		// Output  FromIn       HDMIcon   OutputEn    OSP   Mute
		if strings.ToTitle(res[0]) == "OUTPUT" {
			d.statusReading = ReadingOutput
			d.state.readingOutputNumber = 0
			d.NumOfInputs = len(d.Inputs)
			return
		}

		// 01      Force___11   On
		if len(msg) > 0 {
			res := strings.Fields(msg)
			d.readInputLine(res)
		}
	}

	if d.statusReading == ReadingOutput {
		// 01      01           Off       Yes         SNK   Off
		if len(msg) > 0 {
			res := strings.Fields(msg)
			d.readOutputLine(res)
		}
	}
}

// setModelState will attempt to set the model of the device
func (d *BlustreamMatrix) setModelState(msg string) {
	modelSplit := strings.Split(msg, "Status")
	model := strings.TrimSpace(modelSplit[0])
	model = strings.Replace(model, "HDMI ", "", 1)

	d.state.ModelName = model
	if !strings.Contains(model, "Blustream") {
		d.state.Model = "Blustream " + model
	} else {
		d.state.Model = model
	}

	d.statusReading = ReadingVersion
	d.Driver.Name = d.state.Model
	d.setModel()
}

// setFirmwareState will attempt to find the FW Version of the device.
func (d *BlustreamMatrix) setFirmwareState(msg string) {
	versionSplit := strings.Split(msg, ": ")
	if len(versionSplit) > 1 {
		version := strings.TrimSpace(versionSplit[1])
		d.state.CurrentVersion = version
		d.setModel()
	}

	d.statusReading = WaitingInput
}

// setModel will change the driverName based on the model retrieved from the status
func (d *BlustreamMatrix) setModel() {
	if d.modelSet {
		return
	}

	if len(d.state.Model) > 0 {
		d.Driver.Name = d.state.Model
		if len(d.state.CurrentVersion) > 0 {
			d.Driver.Name = d.state.Model + " v" + d.state.CurrentVersion
			d.modelSet = true
		}

		log.Println("[blustream]: New driver name is: " + d.Driver.Name)
	}
}

// isActive is a helper function to turn "yes", "on" to true when reading status.
func isActive(a string) bool {
	status := strings.ToLower(a)
	if status == "on" || status == "yes" {
		return true
	}
	return false
}

// readInputLine creates multiple BlustreamInput from inputs that are retrieved from the status.
func (d *BlustreamMatrix) readInputLine(res []string) {
	i, err := strconv.Atoi(res[0])
	if err == nil {
		if d.state.readingInputNumber == 0 {
			d.state.readingInputNumber++
		}

		if i == d.state.readingInputNumber {
			if len(res) != 3 {
				return
			}

			if d.NumOfInputs < d.state.readingInputNumber {
				d.state.readingInputNumber++
				newInput := BlustreamInput{
					Input: &drivers.Input{
						InputName: res[0],
						Active:    isActive(res[2]),
					},
					Edid:  res[1],
				}

				d.Inputs = append(d.Inputs, newInput)
			} else {
				for k, input := range d.Inputs {
					if input.InputName == res[0] {
						updatedInput := false
						isSourceActive := isActive(res[2])

						if input.Active != isSourceActive {
							updatedInput = true
							input.Active = isSourceActive
						}

						if input.Edid != res[1] {
							updatedInput = true
							input.Edid = res[1]
						}

						if updatedInput {
							debugLog(input.InputName, "has changed")
							d.updatedInputs = append(d.updatedInputs, k)
						}
					}
				}

				d.state.readingInputNumber++
			}
		}
	}
}

func (d *BlustreamMatrix) readOutputLine(res []string) {
	i, err := strconv.Atoi(res[0])
	if err == nil {
		if d.state.readingOutputNumber == 0 {
			d.state.readingOutputNumber++
		}

		if i == d.state.readingOutputNumber {
			if len(res) != 6 {
				return
			}

			if d.NumOfOutputs < d.state.readingOutputNumber {
				d.state.readingOutputNumber++
				input := d.GetInput(res[1])
				newOutput := BlustreamOutput{
					Output: &drivers.Output{
						OutputName: res[0],
						Active: isActive(res[2]) && isActive(res[3]),
						Input: input,
					},
					Edid:  res[1],
				}

				d.Outputs = append(d.Outputs, newOutput)
			} else {
				for k, output := range d.Outputs {
					if output.OutputName == res[0] {
						updatedInput := false
						isSourceActive := isActive(res[2])

						if output.Active != isSourceActive {
							updatedInput = true
							output.Active = isSourceActive
						}

						if output.Edid != res[1] {
							updatedInput = true
							output.Edid = res[1]
						}

						if updatedInput {
							log.Println(output.OutputName, "has changed")
							d.updatedInputs = append(d.updatedInputs, k)
						}
					}
				}

				d.state.readingOutputNumber++
			}
		}
	}
}