# /driverStatus
```
curl http://localhost:8787/driverStatus | jq
```

```json
{
  "Drivers": [
    {
      "DriverInterface": null,
      "Name": "Startech.com SV431DVIUDDMH2K B4.1",
      "ShortName": "kvm",
      "Inputs": null,
      "Outputs": null,
      "OutputSingle": null,
      "StartAttempted": true,
      "HasError": false,
      "Error": null,
      "NumOfInputs": 4,
      "NumOfOutputs": 1
    },
    {
      "DriverInterface": null,
      "Name": "Blustream CMX44AB v2.22",
      "ShortName": "matrix",
      "OutputMatrix": null,
      "StartAttempted": true,
      "HasError": false,
      "Error": null,
      "NumOfInputs": 4,
      "NumOfOutputs": 4,
      "Inputs": [
        {
          "DriverInput": null,
          "InputName": "01",
          "Active": true,
          "Edid": "Force___11"
        },
        {
          "DriverInput": null,
          "InputName": "02",
          "Active": true,
          "Edid": "Force___11"
        },
        {
          "DriverInput": null,
          "InputName": "03",
          "Active": false,
          "Edid": "Force___11"
        },
        {
          "DriverInput": null,
          "InputName": "04",
          "Active": false,
          "Edid": "Force___11"
        }
      ],
      "Outputs": [
        {
          "DriverOutput": null,
          "OutputName": "01",
          "Active": true,
          "Input": {
            "DriverInput": null,
            "InputName": "01",
            "Active": true,
            "Edid": "Force___11"
          },
          "Edid": "01"
        },
        {
          "DriverOutput": null,
          "OutputName": "02",
          "Active": true,
          "Input": {
            "DriverInput": null,
            "InputName": "02",
            "Active": true,
            "Edid": "Force___11"
          },
          "Edid": "02"
        },
        {
          "DriverOutput": null,
          "OutputName": "03",
          "Active": true,
          "Input": {
            "DriverInput": null,
            "InputName": "02",
            "Active": true,
            "Edid": "Force___11"
          },
          "Edid": "03"
        },
        {
          "DriverOutput": null,
          "OutputName": "04",
          "Active": true,
          "Input": {
            "DriverInput": null,
            "InputName": "02",
            "Active": true,
            "Edid": "Force___11"
          },
          "Edid": "04"
        }
      ]
    }
  ]
}
```

`Drivers` contains an array of all the loaded drivers that the Fence server is using to communicate.

The `Inputs` array describes an input on the matrix (ie, a HDMI connection). `InputName` and `OutputName` are the
names of the inputs/outputs.

`HasError` and `Error` describes the current state of the driver.

The `Output` describes what `Input` (if any) the output is currently connected to.

In the above example: 
* Output `HDMI 1` is showing Input `HDMI 1`.
* Output `HDMI 2`, `HDMI 3` and `HDMI 4` is showing Input `HDMI 2`.

An input not being `Active` may mean that the device is not currently plugged in or powered on.

# /layout
This endpoint dumps a JSON representation of the computed layout.

# /refreshStatus
For any device (currently just the Blustream) that is supported, we will pull the latest output information from the
device.
