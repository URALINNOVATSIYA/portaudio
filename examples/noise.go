package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	pa "github.com/URALINNOVATSIYA/portaudio"
)

func main() {
	check(pa.Initialize())

	noise()

	check(pa.Terminate())
}

func noise() {
	params := pa.HighLatencyParameters(nil, pa.DefaultOutputDevice())
	stream, err := pa.OpenStream(
		params,
		func(s *pa.Stream) pa.StreamCallbackResult {
			out := s.Out()
			for i := range s.Out() {
				out[i] = byte(rand.Uint32())
			}
			return pa.Continue
		},
		func(s *pa.Stream) {
			fmt.Println("Stream is finished!")
		},
	)
	check(err)
	check(stream.Start())
	defer stream.Close()
	time.Sleep(time.Second)
	check(stream.Stop())
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
