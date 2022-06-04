# Fence
## What is Fence?
Fence allows you to automate switching hardware devices when the mouse moves
to the edge of the screen. The functionality is similar to Synergy, but we don't
pass through mouse clicks and key presses (yet?).

Sometimes also known as 'Glide and Switch', the Fence client runs on each PC connected to external
devices through the Fence server. When the mouse moves to the edge of the screen, the client tells
the server the name of the device (ie, the computer name), and the direction of the switch
(left, right, top, bottom). The server then calculates the actions that needs to be performed.

Both the server and the client are cross-platform.

This gives you the convenience of Synergy (move your mouse to the edge of the screen
to swap the computer receiving the inputs) with the stability of using USB DDM
to redirect input devices. Lower latency, better(ish) security and much better
application compatibility.

As an additional benefit, Fence allows you to design your switching on a per-direction, per-device level,
allowing full control over your switching layout.

## Where does it run?
* macOS 11
* Windows 10
* Archlinux (with X11 - not Wayland)

## What devices can be controlled?
* KVM controllers (such as the Startech SV431DVIUDDM or ConnectPRO UDP2-14AP)
* HDMI matrix switches (such as the Blustream CMX44AB)

## How do I get up and running?
Currently, the configuration is hardcoded into the binary at build time.

### Step 1: Configure a server
A Fence server is required to run on a machine that has access to all of the devices that
are wanted to be controlled.

A Raspberry Pi or similar low-powered device is ideal for this.

For my setup, I use a SV431DVIUDDM and CMX44AB.

* Update the `registerDrivers()` function inside `server/main.go` to register drivers for the devices you want to
  control.
* Update the serial port in `server/drivers/startech_kvm/startech_kvm.go`
  and/or `server/drivers/blustream/blustream.go`
* Define the correct layout in `server/layout.go` describing what you want performed when the mouse moves between
  screens

Note that multiple instances of the matrix and KVM drivers can be started at the same time (see `server/main.go`),
allowing for chains if control of a larger range of devices at once is desired.

Start the server:
```shell
# cd server
# go run -addr :8787
# ./server -addr :8787
2022/05/29 13:03:23 Started driver: Startech SV431DVIUDDM
2022/05/29 13:03:23 Started driver: Blustream
2022/05/29 13:03:23 [startech_kvm] Command #1/1: ERROR
2022/05/29 13:03:23 Ignore the first error, we are just initializing our state - looks like this device is correct
2022/05/29 13:03:23 [blustream]: New driver name is: Blustream CMX44AB
2022/05/29 13:03:23 [blustream]: New driver name is: Blustream CMX44AB v2.22
2022/05/29 13:03:24 [startech_kvm] Command #1/1: SV431DVIUDDM F/W Version :H2K B4.1
2022/05/29 13:03:24 [startech_kvm]: New driver name is: Startech.com SV431DVIUDDMH2K B4.1
```

To see if the server is running successfully, you can use one of the [API Endpoints](docs/API_Endpoints.md)

### Step 2: Install client on all machines
```shell
# cd client
# go run -addr 10.2.2.2:8787 -name left
2022/05/29 13:06:23 [GUI] The application has booted
2022/05/29 13:06:23 [Websocket] Starting Client
2022/05/29 13:06:23 [Glide] get screen size:  3360x1890
```

## TODO/Upcoming Features
* [ ] Fix the client, so that it reconnects when the server's connection goes away.
* [ ] Enable using hotkeys to lock the current screen/not send glide commands to the server.
* [ ] Use DDC/CI to control the monitor inputs, so a hardware matrix is not required.
* [ ] Support Synergy, so DDC/CI commands can be issued for additional hardware-free solution.
* [ ] Web interface to define the edges of different devices.
* [ ] Improve switching solution for Blustream devices.
