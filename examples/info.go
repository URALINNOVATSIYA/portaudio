package main

import (
	"encoding/json"
	"fmt"
	"log"

	pa "github.com/URALINNOVATSIYA/portaudio"
)

func main() {
	check(pa.Initialize())

	fmt.Printf("Version number: %d\n", pa.VersionNumber())
	fmt.Printf("Version text: %s\n", pa.VersionText())
	fmt.Printf("Version info: %#v\n", toString(pa.Version()))
	fmt.Printf("Device count: %d\n", pa.DeviceCount())
	fmt.Printf("Default input device: %d\n", pa.DefaultInputDeviceIndex())
	fmt.Printf("Default output device: %d\n", pa.DefaultOutputDeviceIndex())
	fmt.Printf("Default input device info: %#v\n", toString(pa.DefaultInputDevice()))
	fmt.Printf("Default output device info: %#v\n", toString(pa.DefaultOutputDevice()))
	fmt.Printf("Host API count: %d\n", pa.HostApiCount())
	fmt.Printf("Default host api: %d\n", pa.DefaultHostApiIndex())
	fmt.Printf("Default host api info: %#v\n", toString(pa.DefaultHostApi()))
	fmt.Printf("Sample size of formt Int32: %d\n", pa.SampleSize(pa.Int32))

	params := pa.DefaultLowLatencyParameters()
	fmt.Printf("Format supported (%d Hz): %t\n", int(params.SampleRate), pa.IsFormatSupported(params))
	params.SampleRate = 10
	fmt.Printf("Format supported (%d Hz): %t\n", int(params.SampleRate), pa.IsFormatSupported(params))

	check(pa.Terminate())
}

func toString(v any) string {
	json, _ := json.Marshal(v)
	return string(json)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
