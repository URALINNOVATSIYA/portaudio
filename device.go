package portaudio

/*
#cgo pkg-config: portaudio-2.0
#include <portaudio.h>
*/
import "C"
import "time"

type DeviceInfo struct {
	Index                    int
	Name                     string
	MaxInputChannels         int
	MaxOutputChannels        int
	DefaultLowInputLatency   time.Duration
	DefaultLowOutputLatency  time.Duration
	DefaultHighInputLatency  time.Duration
	DefaultHighOutputLatency time.Duration
	DefaultSampleRate        float64
	HostApi                  *HostApiInfo
}

// Device returns a pointer to a DeviceInfo structure containing information about the specified device.
// If the device parameter is out of range the function returns nil.
func Device(index int) *DeviceInfo {
	info := C.Pa_GetDeviceInfo(C.PaDeviceIndex(index))
	if info == nil {
		return nil
	}
	return &DeviceInfo{
		Index:                    index,
		Name:                     C.GoString(info.name),
		MaxInputChannels:         int(info.maxInputChannels),
		MaxOutputChannels:        int(info.maxOutputChannels),
		DefaultLowInputLatency:   duration(info.defaultLowInputLatency),
		DefaultLowOutputLatency:  duration(info.defaultLowOutputLatency),
		DefaultHighInputLatency:  duration(info.defaultHighInputLatency),
		DefaultHighOutputLatency: duration(info.defaultHighOutputLatency),
		DefaultSampleRate:        float64(info.defaultSampleRate),
		HostApi:                  HostApi(int(info.hostApi)),
	}
}

// DeviceCount returns the number of available devices.
// The number of available devices may be zero.
func DeviceCount() int {
	return int(C.Pa_GetDeviceCount())
}

// DefaultInputDeviceIndex returns the index of the default input device.
// The result can be used in the inputDevice parameter to OpenStream().
func DefaultInputDeviceIndex() int {
	return int(C.Pa_GetDefaultInputDevice())
}

// DefaultOutputDeviceIndex returns the index of the default output device.
// The result can be used in the outputDevice parameter to OpenStream().
func DefaultOutputDeviceIndex() int {
	return int(C.Pa_GetDefaultOutputDevice())
}

// DefaultInputDevice returns information about the default input device
func DefaultInputDevice() *DeviceInfo {
	return Device(DefaultInputDeviceIndex())
}

// DefaultOutputDevice returns information about the default output device
func DefaultOutputDevice() *DeviceInfo {
	return Device(DefaultOutputDeviceIndex())
}
