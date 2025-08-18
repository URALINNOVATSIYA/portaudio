package portaudio

/*
#cgo pkg-config: portaudio-2.0
#include <portaudio.h>
extern PaStreamCallback* paStreamCallback;
*/
import "C"
import (
	"runtime/cgo"
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

type StreamCallback = func(
	in, out []byte,
	frames int,
	timeInfo StreamCallbackTimeInfo,
	statusFlags StreamCallbackFlags,
) StreamCallbackResult

// StreamCallbackTimeInfo contains timing information for the
// buffers passed to the stream callback.
type StreamCallbackTimeInfo struct {
	InputBufferAdcTime  time.Duration
	CurrentTime         time.Duration
	OutputBufferDacTime time.Duration
}

// StreamCallbackFlags are flag bit constants for the statusFlags to StreamCallback.
type StreamCallbackFlags C.PaStreamCallbackFlags

// PortAudio stream callback flags.
const (
	// In a stream opened with FramesPerBufferUnspecified,
	// InputUnderflow indicates that input data is all silence (zeros)
	// because no real data is available.
	//
	// In a stream opened without FramesPerBufferUnspecified,
	// InputUnderflow indicates that one or more zero samples have been inserted
	// into the input buffer to compensate for an input underflow.
	InputUnderflow StreamCallbackFlags = C.paInputUnderflow

	// In a stream opened with FramesPerBufferUnspecified,
	// indicates that data prior to the first sample of the
	// input buffer was discarded due to an overflow, possibly
	// because the stream callback is using too much CPU time.
	//
	// Otherwise indicates that data prior to one or more samples
	// in the input buffer was discarded.
	InputOverflow StreamCallbackFlags = C.paInputOverflow

	// Indicates that output data (or a gap) was inserted,
	// possibly because the stream callback is using too much CPU time.
	OutputUnderflow StreamCallbackFlags = C.paOutputUnderflow

	// Indicates that output data will be discarded because no room is available.
	OutputOverflow StreamCallbackFlags = C.paOutputOverflow

	// Some of all of the output data will be used to prime the stream,
	// input data may be zero.
	PrimingOutput StreamCallbackFlags = C.paPrimingOutput
)

// Stream callback result
type StreamCallbackResult = C.PaStreamCallbackResult

const (
	Continue StreamCallbackResult = C.paContinue
	Complete StreamCallbackResult = C.paComplete
	Abort    StreamCallbackResult = C.paAbort
)

// StreamDeviceParameters specifies parameters for
// one device (either input or output) in a stream.
// A nil Device indicates that no device is to be used
// -- i.e., for an input- or output-only stream.
type StreamDeviceParameters struct {
	Device           *DeviceInfo
	ChannelCount     int
	SuggestedLatency time.Duration
}

func (p StreamDeviceParameters) Exists() bool {
	return p.Device != nil
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

type StreamInfo struct {
	InputLatency, OutputLatency time.Duration
	SampleRate                  float64
}

type Stream struct {
	paStream unsafe.Pointer
	params   *StreamParameters
	in, out  []byte
	inSize   int
	outSize  int
	cb       StreamCallback
}

func newStream(params *StreamParameters) *Stream {
	return &Stream{
		params: params,
	}
}

// CpuLoad returns CPU usage information for the stream.
// The "CPU Load" is a fraction of total CPU time consumed by
// a callback stream's audio processing routines including,
// but not limited to the client supplied stream callback.
// This function does not work with blocking read/write streams.
func (s *Stream) CpuLoad() float64 {
	return float64(C.Pa_GetStreamCpuLoad(s.paStream))
}

// Time returns the current time in seconds for a lifespan of a stream.
// Starting and stopping the stream does not affect the passage of time.
func (s *Stream) Time() time.Duration {
	return duration(C.Pa_GetStreamTime(s.paStream))
}

// Info returns information about the stream.
func (s *Stream) Info() *StreamInfo {
	info := C.Pa_GetStreamInfo(s.paStream)
	if info == nil {
		return nil
	}
	return &StreamInfo{
		duration(info.inputLatency),
		duration(info.outputLatency),
		float64(info.sampleRate),
	}
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

// Abort terminates audio processing immediately without waiting for pending buffers to complete.
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

func (s *Stream) Read() ([]byte, error) {
	if s.cb != nil {
		return s.in, nil
	}
	err := goError(C.Pa_ReadStream(s.paStream, unsafe.Pointer(&s.in[0]), C.ulong(s.params.FramesPerBuffer)))
	if err != nil {
		return nil, err
	}
	return s.in, nil
}

func (s *Stream) Write(data []byte) error {
	copy(s.out, data[:len(s.out)])
	if s.cb != nil {
		return nil
	}
	err := goError(C.Pa_WriteStream(s.paStream, unsafe.Pointer(&s.out[0]), C.ulong(s.params.FramesPerBuffer)))
	if err != nil {
		if err == OutputUnderflowed {
			return nil
		}
		return err
	}
	return nil
}

func (s *Stream) Callback() StreamCallback {
	return s.cb
}

func OpenStream(params *StreamParameters, cb StreamCallback) (*Stream, error) {
	s := newStream(params)
	inParams := paStreamParameters(params.Input, params.SampleFormat)
	outParams := paStreamParameters(params.Output, params.SampleFormat)
	scb := C.paStreamCallback
	if cb == nil {
		scb = nil
	} else {
		s.cb = cb
	}
	sptr := cgo.NewHandle(s)
	err := goError(
		C.Pa_OpenStream(
			&s.paStream,
			inParams,
			outParams,
			C.double(params.SampleRate),
			C.ulong(params.FramesPerBuffer),
			C.PaStreamFlags(params.Flags),
			scb,
			unsafe.Pointer(&sptr),
		),
	)
	if err != nil {
		return nil, err
	}
	sampleSize := SampleSize(params.SampleFormat)
	if cb != nil {
		s.inSize = sampleSize * params.Input.ChannelCount
		s.outSize = sampleSize * params.Output.ChannelCount
		return s, nil
	}
	size := sampleSize * int(params.FramesPerBuffer)
	if params.Input.Exists() {
		s.in = make([]byte, size*params.Input.ChannelCount)
	}
	if params.Output.Exists() {
		s.out = make([]byte, size*params.Output.ChannelCount)
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

//export streamCallback
func streamCallback(
	inputBuffer, outputBuffer unsafe.Pointer,
	frames C.ulong,
	timeInfo *C.PaStreamCallbackTimeInfo,
	statusFlags C.PaStreamCallbackFlags,
	userData unsafe.Pointer,
) C.PaStreamCallbackResult {
	size := int(frames)
	s := *(*cgo.Handle)(userData).Value().(*Stream)
	s.in = unsafe.Slice((*byte)(inputBuffer), size * s.inSize)
	s.out = unsafe.Slice((*byte)(outputBuffer), size * s.outSize)
	return s.cb(
		s.in, s.out,
		size,
		StreamCallbackTimeInfo{
			duration(timeInfo.inputBufferAdcTime),
			duration(timeInfo.currentTime),
			duration(timeInfo.outputBufferDacTime),
		},
		StreamCallbackFlags(statusFlags),
	)
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
		p.ChannelCount = min(in.MaxInputChannels, 1)
		p.SuggestedLatency = in.DefaultHighInputLatency
		sampleRate = in.DefaultSampleRate
	}
	if out != nil {
		p := &params.Output
		p.Device = out
		p.ChannelCount = min(out.MaxOutputChannels, 2)
		p.SuggestedLatency = out.DefaultHighOutputLatency
		if r := out.DefaultSampleRate; r < sampleRate || sampleRate == 0 {
			sampleRate = r
		}
	}
	params.SampleRate = sampleRate
	params.FramesPerBuffer = FramesPerBufferUnspecified
	params.SampleFormat = Float32
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
		p.ChannelCount = min(in.MaxInputChannels, 1)
		p.SuggestedLatency = in.DefaultLowInputLatency
		sampleRate = in.DefaultSampleRate
	}
	if out != nil {
		p := &params.Output
		p.Device = out
		p.ChannelCount = min(out.MaxOutputChannels, 2)
		p.SuggestedLatency = out.DefaultLowOutputLatency
		if r := out.DefaultSampleRate; r > sampleRate {
			sampleRate = r
		}
	}
	params.SampleRate = sampleRate
	params.FramesPerBuffer = FramesPerBufferUnspecified
	params.SampleFormat = Float32
	return params
}

func paStreamParameters(p StreamDeviceParameters, sampleFormat SampleFormat) *C.PaStreamParameters {
	if !p.Exists() {
		return nil
	}
	return &C.PaStreamParameters{
		device:           C.int(p.Device.Index),
		channelCount:     C.int(p.ChannelCount),
		sampleFormat:     C.PaSampleFormat(sampleFormat),
		suggestedLatency: C.PaTime(p.SuggestedLatency.Seconds()),
	}
}
