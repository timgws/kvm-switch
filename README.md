# Fence
## What is Fence?
Fence allows you to automate switching hardware devices when the mouse moves
to the edge of the screen. The functionality is similar to Synergy, but we don't
pass through mouse clicks and key presses (yet?).

Both the server and the client are cross-platform.

This gives you the convenience of Synergy (move your mouse to the edge of the screen
to swap the computer receiving the inputs) with the stability of using USB DDM
to redirect input devices. Lower latency, better(ish) security and much better
application compatibility.

## Where does it run?
* macOS 11
* Windows 10
* Archlinux (with X11 - not Wayland)

## What devices can be controlled?
* KVM controllers (such as the Startech SV431DVIUDDM)
* HDMI matrix switches (such as the Blustream CMX44AB)

