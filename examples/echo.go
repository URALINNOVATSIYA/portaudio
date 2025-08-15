package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	pa "github.com/URALINNOVATSIYA/portaudio"
)

func main() {
	err := pa.Initialize()
	check(err)

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
	fmt.Printf("Sample size of formt Int32: %d", pa.SampleSize(pa.Int32))

	fmt.Println("Sync Echo")
	echoSync()
	fmt.Println("Async Echo")
	echoAsync()

	err = pa.Terminate()
	check(err)
}

func toString(v any) string {
	json, _ := json.Marshal(v)
	return string(json)
}

func echoSync() {
	inputDevice := pa.DefaultInputDevice()
	outputDevice := pa.DefaultOutputDevice()
	channelCount := min(inputDevice.MaxInputChannels, outputDevice.MaxOutputChannels)
	params := &pa.StreamParameters{
		Input: pa.StreamDeviceParameters{
			Device:           inputDevice,
			ChannelCount:     channelCount,
			SuggestedLatency: inputDevice.DefaultHighInputLatency,
		},
		Output: pa.StreamDeviceParameters{
			Device:           outputDevice,
			ChannelCount:     channelCount,
			SuggestedLatency: outputDevice.DefaultHighOutputLatency,
		},
		SampleRate:      inputDevice.DefaultSampleRate,
		SampleFormat:    pa.Int16,
		FramesPerBuffer: 512,
		Flags:           pa.ClipOff,
	}
	stream, err := pa.OpenStream(params)
	check(err)
	err = stream.Start()
	check(err)
	var sampleBlock []byte
	const seconds = 15
	for i, max := 0, int(seconds*params.SampleRate/float64(params.FramesPerBuffer)); i < max; i++ {
		sampleBlock, err = stream.Read()
		check(err)
		err = stream.Write(sampleBlock)
		check(err)
	}
	stream.Stop()
}

func echoAsync() {
	params := pa.HighLatencyParameters(pa.DefaultInputDevice(), pa.DefaultOutputDevice())
	stream, err := pa.OpenStream(
		params,
		func(in, out []byte) {
			for i := range in {
				out[i] = in[i]
			}
		},
	)
	check(err)
	err = stream.Start()
	check(err)
	time.Sleep(15 * time.Second)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
