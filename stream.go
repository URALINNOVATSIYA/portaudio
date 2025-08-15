package portaudio

/*
#cgo pkg-config: portaudio-2.0
#include <portaudio.h>
*/
import "C"
import (
	"time"
	"unsafe"
)

type StreamFlags C.PaStreamFlags

const (
	NoFlag                                StreamFlags = C.paNoFlag
	ClipOff                               StreamFlags = C.paClipOff
	DitherOff                             StreamFlags = C.paDitherOff
	NeverDropInput                        StreamFlags = C.paNeverDropInput
	PrimeOutputBuffersUsingStreamCallback StreamFlags = C.paPrimeOutputBuffersUsingStreamCallback
	PlatformSpecificFlags                 StreamFlags = C.paPlatformSpecificFlags
)

type SampleFormat C.PaSampleFormat

const (
	Float32 SampleFormat = C.paFloat32
	Int32   SampleFormat = C.paInt32
	Int24   SampleFormat = C.paInt24
	Int16   SampleFormat = C.paInt16
	Int8    SampleFormat = C.paInt8
	UInt8   SampleFormat = C.paUInt8
	Byte    SampleFormat = C.paUInt8
)

const FramesPerBufferUnspecified = C.paFramesPerBufferUnspecified

// StreamDeviceParameters specifies parameters for
// one device (either input or output) in a stream.
// A nil Device indicates that no device is to be used
// -- i.e., for an input- or output-only stream.
type StreamDeviceParameters struct {
	Device           *DeviceInfo
	ChannelCount     int
	SuggestedLatency time.Duration
}

// StreamParameters includes all parameters required to
// open a stream except for the callback or buffers.
type StreamParameters struct {
	Input, Output   StreamDeviceParameters
	SampleRate      float64
	SampleFormat    SampleFormat
	FramesPerBuffer uint64
	Flags           StreamFlags
}

type Stream struct {
	paStream unsafe.Pointer
	params   *StreamParameters
	in, out  []byte
}

func newStream(params *StreamParameters) *Stream {
	return &Stream{
		params: params,
	}
}

// GetCpuLoad returns CPU usage information for the stream.
// The "CPU Load" is a fraction of total CPU time consumed by
// a callback stream's audio processing routines including,
// but not limited to the client supplied stream callback.
// This function does not work with blocking read/write streams.
func (s *Stream) GetCpuLoad() float64 {
	return float64(C.Pa_GetStreamCpuLoad(s.paStream))
}

// Start commences audio processing.
func (s *Stream) Start() error {
	return goError(C.Pa_StartStream(s.paStream))
}

// Stop terminates audio processing.
// It waits until all pending audio buffers have been played before it returns.
func (s *Stream) Stop() error {
	return goError(C.Pa_StopStream(s.paStream))
}

// Close closes an audio stream. If the audio stream is active it discards any pending buffers.
func (s *Stream) Close() error {
	return goError(C.Pa_CloseStream(s.paStream))
}

func (s *Stream) Abort() error {
	return goError(C.Pa_AbortStream(s.paStream))
}

// ReadAvailable returns the number of frames that can be read from the stream without waiting.
func (s *Stream) ReadAvailable() (int, error) {
	size := C.Pa_GetStreamReadAvailable(s.paStream)
	if size < 0 {
		return 0, goError(C.PaError(size))
	}
	return int(size), nil
}

// WriteAvailable returns the number of frames that can be written from the stream without waiting.
func (s *Stream) WriteAvailable() (int, error) {
	size := C.Pa_GetStreamWriteAvailable(s.paStream)
	if size < 0 {
		return 0, goError(C.PaError(size))
	}
	return int(size), nil
}

func (s *Stream) Read(ptr unsafe.Pointer) error {
	err := goError(C.Pa_ReadStream(s.paStream, ptr, C.ulong(s.params.FramesPerBuffer)))
	if err != nil {
		return err
	}
	return nil
}

func (s *Stream) Write(ptr unsafe.Pointer) error {
	err := goError(C.Pa_WriteStream(s.paStream, ptr, C.ulong(s.params.FramesPerBuffer)))
	if err != nil {
		return err
	}
	return nil
}

func OpenStream(
	params *StreamParameters,
) (*Stream, error) {
	s := newStream(params)
	inParams := paStreamParameters(params.Input, params.SampleFormat)
	outParams := paStreamParameters(params.Output, params.SampleFormat)
	err := goError(
		C.Pa_OpenStream(
			&s.paStream,
			inParams,
			outParams,
			C.double(params.SampleRate),
			C.ulong(params.FramesPerBuffer),
			C.PaStreamFlags(params.Flags),
			nil,
			nil,
		),
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// OpenDefaultStream is a simplified version of OpenStream() that opens the default input and/or output devices.
func OpenDefaultStream(
	numInputChannels, numOutputChannels int,
	sampleFormat SampleFormat,
	sampleRate float64,
	framesPerBuffer uint64,
) (*Stream, error) {
	//s := newStream()

	return nil, nil
}

// SampleSize returns the size of a given sample format in bytes or 0 on error.
func SampleSize(format SampleFormat) int {
	if size := C.Pa_GetSampleSize(C.PaSampleFormat(format)); size >= 0 {
		return int(size)
	}
	return 0
}

// HighLatencyParameters are mono in, stereo out (if supported),
// high latency, the smaller of the default sample rates of the two devices,
// and FramesPerBufferUnspecified. One of the devices may be nil.
func HighLatencyParameters(in, out *DeviceInfo) *StreamParameters {
	params := &StreamParameters{}
	sampleRate := 0.0
	if in != nil {
		p := &params.Input
		p.Device = in
		p.ChannelCount = 1
		if in.MaxInputChannels < 1 {
			p.ChannelCount = in.MaxInputChannels
		}
		p.SuggestedLatency = in.DefaultHighInputLatency
		sampleRate = in.DefaultSampleRate
	}
	if out != nil {
		p := &params.Output
		p.Device = out
		p.ChannelCount = 2
		if out.MaxOutputChannels < 2 {
			p.ChannelCount = out.MaxOutputChannels
		}
		p.SuggestedLatency = out.DefaultHighOutputLatency
		if r := out.DefaultSampleRate; r < sampleRate || sampleRate == 0 {
			sampleRate = r
		}
	}
	params.SampleRate = sampleRate
	params.FramesPerBuffer = FramesPerBufferUnspecified
	return params
}

// LowLatencyParameters are mono in, stereo out (if supported),
// low latency, the larger of the default sample rates of the two devices,
// and FramesPerBufferUnspecified. One of the devices may be nil.
func LowLatencyParameters(in, out *DeviceInfo) *StreamParameters {
	params := &StreamParameters{}
	sampleRate := 0.0
	if in != nil {
		p := &params.Input
		p.Device = in
		p.ChannelCount = 1
		if in.MaxInputChannels < 1 {
			p.ChannelCount = in.MaxInputChannels
		}
		p.SuggestedLatency = in.DefaultLowInputLatency
		sampleRate = in.DefaultSampleRate
	}
	if out != nil {
		p := &params.Output
		p.Device = out
		p.ChannelCount = 2
		if out.MaxOutputChannels < 2 {
			p.ChannelCount = out.MaxOutputChannels
		}
		p.SuggestedLatency = out.DefaultLowOutputLatency
		if r := out.DefaultSampleRate; r > sampleRate {
			sampleRate = r
		}
	}
	params.SampleRate = sampleRate
	params.FramesPerBuffer = FramesPerBufferUnspecified
	return params
}

func paStreamParameters(p StreamDeviceParameters, sampleFormat SampleFormat) *C.PaStreamParameters {
	if p.Device == nil {
		return nil
	}
	return &C.PaStreamParameters{
		device:           C.int(p.Device.Index),
		channelCount:     C.int(p.ChannelCount),
		sampleFormat:     C.PaSampleFormat(sampleFormat),
		suggestedLatency: C.PaTime(p.SuggestedLatency.Seconds()),
	}
}
