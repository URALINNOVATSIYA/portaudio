package main

import (
	"encoding/json"
	"fmt"
	"log"
	"unsafe"

	pa "github.com/URALINNOVATSIYA/portaudio"
)

func main() {
	err := pa.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Version number: %d\n", pa.VersionNumber())
	fmt.Printf("Version text: %s\n", pa.VersionText())
	fmt.Printf("Version info: %#v\n", toString(pa.Version()))
	fmt.Printf("Device count: %d\n", pa.DeviceCount())
	fmt.Printf("Default input device: %d\n", pa.DefaultInputDeviceIndex())
	fmt.Printf("Default output device: %d\n", pa.DefaultOutputDeviceIndex())
	fmt.Printf("Device info: %#v\n", toString(pa.Device(0)))
	fmt.Printf("Host API count: %d\n", pa.HostApiCount())
	fmt.Printf("Default host api: %d\n", pa.DefaultHostApi())
	fmt.Printf("Default host api info: %#v\n", toString(pa.DefaultHostApi()))
	fmt.Printf("Sample size of formt Int32: %d", pa.SampleSize(pa.Int32))

	echo()

	err = pa.Terminate()
	if err != nil {
		log.Fatal(err)
	}
}

func toString(v any) string {
	json, _ := json.Marshal(v)
	return string(json)
}

func echo() {
	inputDevice := pa.DefaultInputDevice()
	outputDevice := pa.DefaultOutputDevice()
	channelCount := inputDevice.MaxInputChannels
	if channelCount > outputDevice.MaxOutputChannels {
		channelCount = outputDevice.MaxOutputChannels
	}
	params := &pa.StreamParameters{
		Input: pa.StreamDeviceParameters{
			Device:   inputDevice,
			ChannelCount: channelCount,
			SuggestedLatency: inputDevice.DefaultHighInputLatency,
		},
		Output: pa.StreamDeviceParameters{
			Device: outputDevice,
			ChannelCount: channelCount,
			SuggestedLatency: outputDevice.DefaultHighOutputLatency,
		},
		SampleRate: inputDevice.DefaultSampleRate,
		SampleFormat: pa.Float32,
		FramesPerBuffer: 512,
		Flags: pa.ClipOff,
	}
	stream, err := pa.OpenStream(params)
	if err != nil {
		log.Fatal(err)
	}
	if err = stream.Start(); err != nil {
		log.Fatal(err)
	}
	sampleBlock := make([]byte, pa.SampleSize(params.SampleFormat) * int(params.FramesPerBuffer) * channelCount)
	ptr := unsafe.Pointer(&sampleBlock[0])
	for {
		size, _ := stream.WriteAvailable()
		if size > 0 {
			break
		}
	}
	for i, max := 0, int(10 * params.SampleRate / float64(params.FramesPerBuffer)); i < max; i++ {
		if err = stream.Write(ptr); err != nil {
			log.Fatal(err)
		}
		if err = stream.Read(ptr); err != nil {
			log.Fatal(err)
		}
	}
}
