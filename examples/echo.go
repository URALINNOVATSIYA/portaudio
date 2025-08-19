package main

import (
	"fmt"
	"log"
	"time"

	pa "github.com/URALINNOVATSIYA/portaudio"
)

func main() {
	check(pa.Initialize())

	fmt.Println("Sync Echo")
	echoSync()
	fmt.Println("Async Echo")
	echoAsync()

	check(pa.Terminate())
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
		SampleFormat:    pa.Float32,
		FramesPerBuffer: 512,
		Flags:           pa.ClipOff,
	}
	stream, err := pa.OpenStream(params, nil, nil)
	check(err)
	check(stream.Start())
	defer stream.Close()
	var sampleBlock []byte
	const seconds = 15
	for i, max := 0, int(seconds*params.SampleRate/float64(params.FramesPerBuffer)); i < max; i++ {
		sampleBlock, err = stream.Read()
		check(err)
		err = stream.Write(sampleBlock)
		check(err)
	}
	check(stream.Stop())
}

func echoAsync() {
	params := pa.DefaultLowLatencyParameters()
	params.Input.ChannelCount = 1
	params.Output.ChannelCount = 1
	stream, err := pa.OpenStream(
		params,
		func(s *pa.Stream) pa.StreamCallbackResult {
			copy(s.Out(), s.In())
			return pa.Continue
		},
		nil,
	)
	check(err)
	check(stream.Start())
	defer stream.Close()
	time.Sleep(15 * time.Second)
	check(stream.Stop())
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
