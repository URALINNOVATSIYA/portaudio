package portaudio

/*
#cgo pkg-config: portaudio-2.0
#include <portaudio.h>
extern PaStreamCallback* paStreamCallback;
extern PaStreamFinishedCallback* paStreamFinishedCallback;
*/
import "C"
import (
	"fmt"
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

func (f SampleFormat) IsInterleaved() bool {
	return f&NonInterleaved == 0
}

func (f SampleFormat) IsNonInterleaved() bool {
	return f&NonInterleaved != 0
}

const (
	NonInterleaved SampleFormat = C.paNonInterleaved
	Float32        SampleFormat = C.paFloat32
	Int32          SampleFormat = C.paInt32
	Int24          SampleFormat = C.paInt24
	Int16          SampleFormat = C.paInt16
	Int8           SampleFormat = C.paInt8
	UInt8          SampleFormat = C.paUInt8
	Byte           SampleFormat = C.paUInt8
)

const FramesPerBufferUnspecified = C.paFramesPerBufferUnspecified

type StreamFinishedCallback = func(*Stream)
type StreamCallback = func(*Stream) StreamCallbackResult

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
	// Signal that the stream should continue invoking the callback and processing audio.
	Continue StreamCallbackResult = C.paContinue
	// Signal that the stream should stop invoking the callback and finish once all output samples have played.
	Complete StreamCallbackResult = C.paComplete
	// Signal that the stream should stop invoking the callback and finish as soon as possible.
	Abort StreamCallbackResult = C.paAbort
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
	paStream         unsafe.Pointer
	params           *StreamParameters
	in, out          []byte
	inS, outS        [][]byte
	inSize, outSize  int
	frameCount       int
	timeInfo         StreamCallbackTimeInfo
	statusFlags      StreamCallbackFlags
	callback         StreamCallback
	finishedCallback StreamFinishedCallback
}

func newStream(params *StreamParameters) *Stream {
	return &Stream{
		params: params,
	}
}

// OpenStream opens a stream for either input, output or both.
func OpenStream(
	params *StreamParameters,
	callback StreamCallback,
	finishedCallback StreamFinishedCallback,
) (*Stream, error) {
	s := newStream(params)
	inParams := paStreamParameters(params.Input, params.SampleFormat)
	outParams := paStreamParameters(params.Output, params.SampleFormat)
	scb := C.paStreamCallback
	if callback == nil {
		scb = nil
	} else {
		s.callback = callback
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
	if finishedCallback != nil {
		if err = s.SetFinishedCallback(finishedCallback); err != nil {
			return nil, err
		}
	}
	sampleSize := SampleSize(params.SampleFormat)
	if callback != nil {
		if params.SampleFormat.IsNonInterleaved() {
			if params.Input.Exists() {
				s.inSize = sampleSize
				s.inS = make([][]byte, params.Input.ChannelCount)
			}
			if params.Output.Exists() {
				s.outSize = sampleSize
				s.outS = make([][]byte, params.Output.ChannelCount)
			}
			return s, nil
		}
		if params.Input.Exists() {
			s.inSize = sampleSize * params.Input.ChannelCount
		}
		if params.Output.Exists() {
			s.outSize = sampleSize * params.Output.ChannelCount
		}
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

// IsActive determines whether the stream is active. A stream is active after
// a successful call to Start(), until it becomes inactive either as
// a result of a call to Stop() or Abort(), or as a result of a return value other
// than Continue from the stream callback. In the latter case, the stream is considered
// inactive after the last buffer has finished playing.
func (s *Stream) IsActive() bool {
	return int(C.Pa_IsStreamActive(s.paStream)) == 1
}

// IsStopped determines whether the stream is stopped. A stream is considered to be stopped
// prior to a successful call to Start() and after a successful call to Stop() or Abort().
// If a stream callback returns a value other than Continue the stream is NOT considered to be stopped.
func (s *Stream) IsStopped() bool {
	return int(C.Pa_IsStreamStopped(s.paStream)) == 1
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

func (s *Stream) FrameCount() int {
	return s.frameCount
}

func (s *Stream) StatusFlags() StreamCallbackFlags {
	return s.statusFlags
}

func (s *Stream) TimeInfo() StreamCallbackTimeInfo {
	return s.timeInfo
}

func (s *Stream) Callback() StreamCallback {
	return s.callback
}

func (s *Stream) FinishedCallback() StreamFinishedCallback {
	return s.finishedCallback
}

func (s *Stream) SetFinishedCallback(callback StreamFinishedCallback) error {
	cb := C.paStreamFinishedCallback
	if callback == nil {
		cb = nil
	} else {
		s.finishedCallback = callback
	}
	return goError(C.Pa_SetStreamFinishedCallback(s.paStream, cb))
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

func (s *Stream) InS() [][]byte {
	return s.inS
}

func (s *Stream) OutS() [][]byte {
	return s.outS
}

func (s *Stream) In() []byte {
	return s.in
}

func (s *Stream) Out() []byte {
	return s.out
}

// Read reads samples from an input stream. The function doesn't return until the entire buffer
// has been filled - this may involve waiting for the operating system to supply the data.
func (s *Stream) Read() ([]byte, error) {
	if s.callback != nil {
		return s.in, nil
	}
	err := goError(C.Pa_ReadStream(s.paStream, unsafe.Pointer(&s.in[0]), C.ulong(s.params.FramesPerBuffer)))
	if err != nil {
		return nil, err
	}
	return s.in, nil
}

// Write writes samples to an output stream. This function doesn't return until the entire buffer
// has been written - this may involve waiting for the operating system to consume the data.
func (s *Stream) Write(data []byte) error {
	copy(s.out, data[:len(s.out)])
	if s.callback != nil {
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

// SampleSize returns the size of a given sample format in bytes or 0 on error.
func SampleSize(format SampleFormat) int {
	if size := C.Pa_GetSampleSize(C.PaSampleFormat(format)); size >= 0 {
		return int(size)
	}
	return 0
}

func DefaultHighLatencyParameters() *StreamParameters {
	return HighLatencyParameters(DefaultInputDevice(), DefaultOutputDevice())
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

func DefaultLowLatencyParameters() *StreamParameters {
	return LowLatencyParameters(DefaultInputDevice(), DefaultOutputDevice())
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

// IsFormatSupported determines whether it would be possible to open a stream with the specified parameters.
// Input device must be nil for output-only streams and
// output device must be nil for input-only streams respectively.
// Returns true if the format is supported, and false otherwise.
func IsFormatSupported(params *StreamParameters) bool {
	return goError(
		C.Pa_IsFormatSupported(
			paStreamParameters(params.Input, params.SampleFormat),
			paStreamParameters(params.Output, params.SampleFormat),
			C.double(params.SampleRate),
		),
	) == nil
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

//export streamCallback
func streamCallback(
	inputBuffer, outputBuffer unsafe.Pointer,
	frameCount C.ulong,
	timeInfo *C.PaStreamCallbackTimeInfo,
	statusFlags C.PaStreamCallbackFlags,
	userData unsafe.Pointer,
) C.PaStreamCallbackResult {
	s := (*cgo.Handle)(userData).Value().(*Stream)
	s.statusFlags = StreamCallbackFlags(statusFlags)
	s.timeInfo = StreamCallbackTimeInfo{
		duration(timeInfo.inputBufferAdcTime),
		duration(timeInfo.currentTime),
		duration(timeInfo.outputBufferDacTime),
	}
	s.frameCount = int(frameCount)
	if s.params.SampleFormat.IsInterleaved() {
		if uintptr(inputBuffer) != 0 {
			s.in = unsafe.Slice((*byte)(inputBuffer), s.frameCount*s.inSize)
		}
		if uintptr(outputBuffer) != 0 {
			s.out = unsafe.Slice((*byte)(outputBuffer), s.frameCount*s.outSize)
		}
	} else {
		size := s.frameCount*s.outSize
		//outPtrs := unsafe.Slice((*unsafe.Pointer)(outputBuffer), s.params.Output.ChannelCount)
		//for i := range outPtrs {
		fmt.Println(*(*unsafe.Pointer)(outputBuffer))
		fmt.Println(*(*unsafe.Pointer)(unsafe.Pointer(uintptr(outputBuffer) + unsafe.Sizeof(uintptr(0)))))
		s.outS[0] = unsafe.Slice((*byte)(*(*unsafe.Pointer)(outputBuffer)), size)
		s.outS[1] = unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(uintptr(outputBuffer) + unsafe.Sizeof(uintptr(0))))), size)
		//}
	}
	return s.callback(s)
}

//export streamFinishedCallback
func streamFinishedCallback(userData unsafe.Pointer) {
	s := (*cgo.Handle)(userData).Value().(*Stream)
	s.finishedCallback(s)
}
