package portaudio

/*
#cgo pkg-config: portaudio-2.0
#include <portaudio.h>
extern PaStreamCallback* paStreamCallback;
extern PaStreamFinishedCallback* paStreamFinishedCallback;
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
)

const FramesPerBufferUnspecified = C.paFramesPerBufferUnspecified

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
	RawOutput       bool
}

type StreamInfo struct {
	InputLatency, OutputLatency time.Duration
	SampleRate                  float64
}

type Stream[T any] struct {
	paStream         unsafe.Pointer
	params           *StreamParameters
	in, out          []T
	inS, outS        [][]T
	inSize, outSize  int
	frameCount       int
	timeInfo         StreamCallbackTimeInfo
	statusFlags      StreamCallbackFlags
	callback         func(*Stream[T]) StreamCallbackResult
	finishedCallback func(*Stream[T])
}

func newStream[T any](params *StreamParameters) *Stream[T] {
	return &Stream[T]{
		params: params,
	}
}

// OpenStream opens a stream for either input, output or both.
func OpenStream[T any](
	params *StreamParameters,
	callback func(*Stream[T]) StreamCallbackResult,
	finishedCallback func(*Stream[T]),
) (*Stream[T], error) {
	s := newStream[T](params)
	return s, s.init(params, callback, finishedCallback)
}

func (s *Stream[T]) init(
	params *StreamParameters,
	callback func(*Stream[T]) StreamCallbackResult,
	finishedCallback func(*Stream[T]),
) error {
	inParams := paStreamParameters(params.Input, params.SampleFormat)
	outParams := paStreamParameters(params.Output, params.SampleFormat)
	scb := C.paStreamCallback
	if callback == nil {
		scb = nil
	} else {
		s.callback = callback
	}
	sid := cgo.NewHandle(s)
	err := goError(
		C.Pa_OpenStream(
			&s.paStream,
			inParams,
			outParams,
			C.double(params.SampleRate),
			C.ulong(params.FramesPerBuffer),
			C.PaStreamFlags(params.Flags),
			scb,
			unsafe.Pointer(sid),
		),
	)
	if err != nil {
		return err
	}
	if finishedCallback != nil {
		if err = s.SetFinishedCallback(finishedCallback); err != nil {
			return err
		}
	}
	sampleSize := 1
	if params.RawOutput {
		sampleSize = SampleSize(params.SampleFormat)
	}
	if callback != nil {
		if params.SampleFormat.IsNonInterleaved() {
			if params.Input.Exists() {
				s.inSize = sampleSize
				s.inS = make([][]T, params.Input.ChannelCount)
			}
			if params.Output.Exists() {
				s.outSize = sampleSize
				s.outS = make([][]T, params.Output.ChannelCount)
			}
			return nil
		}
		if params.Input.Exists() {
			s.inSize = sampleSize * params.Input.ChannelCount
		}
		if params.Output.Exists() {
			s.outSize = sampleSize * params.Output.ChannelCount
		}
		return nil
	}
	size := sampleSize * int(params.FramesPerBuffer)
	if params.Input.Exists() {
		s.in = make([]T, size*params.Input.ChannelCount)
	}
	if params.Output.Exists() {
		s.out = make([]T, size*params.Output.ChannelCount)
	}
	return nil
}

// IsActive determines whether the stream is active. A stream is active after
// a successful call to Start(), until it becomes inactive either as
// a result of a call to Stop() or Abort(), or as a result of a return value other
// than Continue from the stream callback. In the latter case, the stream is considered
// inactive after the last buffer has finished playing.
func (s *Stream[T]) IsActive() bool {
	return int(C.Pa_IsStreamActive(s.paStream)) == 1
}

// IsStopped determines whether the stream is stopped. A stream is considered to be stopped
// prior to a successful call to Start() and after a successful call to Stop() or Abort().
// If a stream callback returns a value other than Continue the stream is NOT considered to be stopped.
func (s *Stream[T]) IsStopped() bool {
	return int(C.Pa_IsStreamStopped(s.paStream)) == 1
}

// CpuLoad returns CPU usage information for the stream.
// The "CPU Load" is a fraction of total CPU time consumed by
// a callback stream's audio processing routines including,
// but not limited to the client supplied stream callback.
// This function does not work with blocking read/write streams.
func (s *Stream[T]) CpuLoad() float64 {
	return float64(C.Pa_GetStreamCpuLoad(s.paStream))
}

// Time returns the current time in seconds for a lifespan of a stream.
// Starting and stopping the stream does not affect the passage of time.
func (s *Stream[T]) Time() time.Duration {
	return duration(C.Pa_GetStreamTime(s.paStream))
}

// Info returns information about the stream.
func (s *Stream[T]) Info() *StreamInfo {
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

func (s *Stream[T]) FrameCount() int {
	return s.frameCount
}

func (s *Stream[T]) StatusFlags() StreamCallbackFlags {
	return s.statusFlags
}

func (s *Stream[T]) TimeInfo() StreamCallbackTimeInfo {
	return s.timeInfo
}

func (s *Stream[T]) SetFinishedCallback(callback func(*Stream[T])) error {
	cb := C.paStreamFinishedCallback
	if callback == nil {
		cb = nil
	} else {
		s.finishedCallback = callback
	}
	return goError(C.Pa_SetStreamFinishedCallback(s.paStream, cb))
}

// Start commences audio processing.
func (s *Stream[T]) Start() error {
	return goError(C.Pa_StartStream(s.paStream))
}

// Stop terminates audio processing.
// It waits until all pending audio buffers have been played before it returns.
func (s *Stream[T]) Stop() error {
	return goError(C.Pa_StopStream(s.paStream))
}

// Close closes an audio stream. If the audio stream is active it discards any pending buffers.
func (s *Stream[T]) Close() error {
	return goError(C.Pa_CloseStream(s.paStream))
}

// Abort terminates audio processing immediately without waiting for pending buffers to complete.
func (s *Stream[T]) Abort() error {
	return goError(C.Pa_AbortStream(s.paStream))
}

// ReadAvailable returns the number of frames that can be read from the stream without waiting.
func (s *Stream[T]) ReadAvailable() (int, error) {
	size := C.Pa_GetStreamReadAvailable(s.paStream)
	if size < 0 {
		return 0, goError(C.PaError(size))
	}
	return int(size), nil
}

// WriteAvailable returns the number of frames that can be written from the stream without waiting.
func (s *Stream[T]) WriteAvailable() (int, error) {
	size := C.Pa_GetStreamWriteAvailable(s.paStream)
	if size < 0 {
		return 0, goError(C.PaError(size))
	}
	return int(size), nil
}

func (s *Stream[T]) In() []T {
	return s.in
}

func (s *Stream[T]) Out() []T {
	return s.out
}

func (s *Stream[T]) InS() [][]T {
	return s.inS
}

func (s *Stream[T]) OutS() [][]T {
	return s.outS
}

// Read reads samples from an input stream. The function doesn't return until the entire buffer
// has been filled - this may involve waiting for the operating system to supply the data.
func (s *Stream[T]) Read() ([]T, error) {
	if s.callback != nil {
		return s.in, nil
	}
	err := goError(C.Pa_ReadStream(s.paStream, unsafe.Pointer(&s.in[0]), C.ulong(s.params.FramesPerBuffer)))
	if err != nil {
		return nil, err
	}
	return s.in, nil
}

func (s *Stream[T]) ReadS() ([][]T, error) {
	if s.callback != nil {
		return s.inS, nil
	}
	err := goError(C.Pa_ReadStream(s.paStream, unsafe.Pointer(&s.inS[0]), C.ulong(s.params.FramesPerBuffer)))
	if err != nil {
		return nil, err
	}
	return s.inS, nil
}

// Write writes samples to an output stream. This function doesn't return until the entire buffer
// has been written - this may involve waiting for the operating system to consume the data.
func (s *Stream[T]) Write(data []T) error {
	copy(s.out, data)
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

func (s *Stream[T]) WriteS(data [][]T) error {
	copy(s.outS, data)
	if s.callback != nil {
		return nil
	}
	err := goError(C.Pa_WriteStream(s.paStream, unsafe.Pointer(&s.outS[0]), C.ulong(s.params.FramesPerBuffer)))
	if err != nil {
		if err == OutputUnderflowed {
			return nil
		}
		return err
	}
	return nil
}

func (s *Stream[T]) Callback(
	in, out unsafe.Pointer,
	frameCount C.ulong,
	timeInfo *C.PaStreamCallbackTimeInfo,
	statusFlags C.PaStreamCallbackFlags,
) StreamCallbackResult {
	s.statusFlags = StreamCallbackFlags(statusFlags)
	s.timeInfo = StreamCallbackTimeInfo{
		duration(timeInfo.inputBufferAdcTime),
		duration(timeInfo.currentTime),
		duration(timeInfo.outputBufferDacTime),
	}
	s.frameCount = int(frameCount)
	s.setInBuffer(in)
	s.setOutBuffer(out)
	return s.callback(s)
}

func (s *Stream[T]) setInBuffer(ptr unsafe.Pointer) {
	size := s.frameCount * s.inSize
	if s.params.SampleFormat.IsInterleaved() {
		s.in = unsafe.Slice((*T)(ptr), size)
		return
	}
	outPtrs := unsafe.Slice((*unsafe.Pointer)(ptr), s.params.Input.ChannelCount)
	for i, innerPtr := range outPtrs {
		s.inS[i] = unsafe.Slice((*T)(innerPtr), size)
	}
}

func (s *Stream[T]) setOutBuffer(ptr unsafe.Pointer) {
	size := s.frameCount * s.outSize
	if s.params.SampleFormat.IsInterleaved() {
		s.out = unsafe.Slice((*T)(ptr), size)
		return
	}
	outPtrs := unsafe.Slice((*unsafe.Pointer)(ptr), s.params.Output.ChannelCount)
	for i, innerPtr := range outPtrs {
		s.outS[i] = unsafe.Slice((*T)(innerPtr), size)
	}
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
	in, out unsafe.Pointer,
	frameCount C.ulong,
	timeInfo *C.PaStreamCallbackTimeInfo,
	statusFlags C.PaStreamCallbackFlags,
	userData unsafe.Pointer,
) C.PaStreamCallbackResult {
	switch s := (cgo.Handle)(userData).Value().(type) {
	case *Stream[byte]:
		return s.Callback(in, out, frameCount, timeInfo, statusFlags)
	case *Stream[int8]:
		return s.Callback(in, out, frameCount, timeInfo, statusFlags)
	case *Stream[int16]:
		return s.Callback(in, out, frameCount, timeInfo, statusFlags)
	case *Stream[uint16]:
		return s.Callback(in, out, frameCount, timeInfo, statusFlags)
	case *Stream[int32]:
		return s.Callback(in, out, frameCount, timeInfo, statusFlags)
	case *Stream[uint32]:
		return s.Callback(in, out, frameCount, timeInfo, statusFlags)
	case *Stream[float32]:
		return s.Callback(in, out, frameCount, timeInfo, statusFlags)
	}
	return Abort
}

//export streamFinishedCallback
func streamFinishedCallback(userData unsafe.Pointer) {
	switch s := (cgo.Handle)(userData).Value().(type) {
	case *Stream[byte]:
		s.finishedCallback(s)
	case *Stream[int8]:
		s.finishedCallback(s)
	case *Stream[int16]:
		s.finishedCallback(s)
	case *Stream[uint16]:
		s.finishedCallback(s)
	case *Stream[int32]:
		s.finishedCallback(s)
	case *Stream[uint32]:
		s.finishedCallback(s)
	case *Stream[float32]:
		s.finishedCallback(s)
	}
}
