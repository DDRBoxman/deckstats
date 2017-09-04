# Deckstats

Display information from OpenHardwareMonitor on an Elgato Streamdeck

![Demo gif](demo.gif)

## Open Hardware Monitor

I actually recommend the fork of Open Hardware Monitor (LibreHardwareMonitor). It has support for newer processors that OHM doesn't.

https://ci.appveyor.com/project/LibreHardwareMonitor/librehardwaremonitor/build/artifacts

## Getting a proper windows dev enviornment

Install msys2

Open the mingw64 terminal (not msys2)

```
pacman -S git mingw-w64-x86_64-gcc pacman -S mingw-w64-x86_64-go
```
