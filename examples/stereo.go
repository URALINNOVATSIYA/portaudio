package main

import (
	"log"
	"math"
	"time"

	pa "github.com/URALINNOVATSIYA/portaudio"
)

type stereoSine struct {
	*pa.Stream
	stepL, phaseL float64
	stepR, phaseR float64
}

func newStereoSine(freqL, freqR float64) *stereoSine {
	var err error
	params := pa.HighLatencyParameters(nil, pa.DefaultOutputDevice())
	params.Output.ChannelCount = 2
	params.SampleFormat = pa.Float32 | pa.NonInterleaved
	s := &stereoSine{nil, freqL / params.SampleRate, 0, freqR / params.SampleRate, 0}
	s.Stream, err = pa.OpenStream(params, s.processAudio, nil)
	check(err)
	return s
}

func (g *stereoSine) processAudio(s *pa.Stream) pa.StreamCallbackResult {
	out := s.OutS()
	for i, max := 0, len(out[0]); i < max; i += 4 {
		x := math.Float32bits(float32(math.Sin(2 * math.Pi * g.phaseL)))
		out[0][i] = byte(x)
		out[0][i+1] = byte(x >> 8)
		out[0][i+2] = byte(x >> 16)
		out[0][i+3] = byte(x >> 24)
		_, g.phaseL = math.Modf(g.phaseL + g.stepL)
		x = math.Float32bits(float32(math.Sin(2 * math.Pi * g.phaseR)))
		out[1][i] = byte(x)
		out[1][i+1] = byte(x >> 8)
		out[1][i+2] = byte(x >> 16)
		out[1][i+3] = byte(x >> 24)
		_, g.phaseR = math.Modf(g.phaseR + g.stepR)
	}
	return pa.Continue
}

func main() {
	check(pa.Initialize())

	playStereo()

	check(pa.Terminate())
}

func playStereo() {
	stream := newStereoSine(128, 320)
	check(stream.Start())
	defer stream.Close()
	time.Sleep(5 * time.Second)
	check(stream.Stop())
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
