package portaudio

/*
#cgo pkg-config: portaudio-2.0
#include <portaudio.h>
*/
import "C"
import (
	"time"
)

// See https://portaudio.com/docs/v19-doxydocs-dev/ for more info about PortAudio

type Error C.PaError

func (err Error) Error() string {
	return C.GoString(C.Pa_GetErrorText(C.PaError(err)))
}

// PortAudio Errors.
const (
	NotInitialized                        Error = C.paNotInitialized
	InvalidChannelCount                   Error = C.paInvalidChannelCount
	InvalidSampleRate                     Error = C.paInvalidSampleRate
	InvalidDevice                         Error = C.paInvalidDevice
	InvalidFlag                           Error = C.paInvalidFlag
	SampleFormatNotSupported              Error = C.paSampleFormatNotSupported
	BadIODeviceCombination                Error = C.paBadIODeviceCombination
	InsufficientMemory                    Error = C.paInsufficientMemory
	BufferTooBig                          Error = C.paBufferTooBig
	BufferTooSmall                        Error = C.paBufferTooSmall
	NullCallback                          Error = C.paNullCallback
	BadStreamPtr                          Error = C.paBadStreamPtr
	TimedOut                              Error = C.paTimedOut
	InternalError                         Error = C.paInternalError
	DeviceUnavailable                     Error = C.paDeviceUnavailable
	IncompatibleHostApiSpecificStreamInfo Error = C.paIncompatibleHostApiSpecificStreamInfo
	StreamIsStopped                       Error = C.paStreamIsStopped
	StreamIsNotStopped                    Error = C.paStreamIsNotStopped
	InputOverflowed                       Error = C.paInputOverflowed
	OutputUnderflowed                     Error = C.paOutputUnderflowed
	HostApiNotFound                       Error = C.paHostApiNotFound
	InvalidHostApi                        Error = C.paInvalidHostApi
	CanNotReadFromACallbackStream         Error = C.paCanNotReadFromACallbackStream
	CanNotWriteToACallbackStream          Error = C.paCanNotWriteToACallbackStream
	CanNotReadFromAnOutputOnlyStream      Error = C.paCanNotReadFromAnOutputOnlyStream
	CanNotWriteToAnInputOnlyStream        Error = C.paCanNotWriteToAnInputOnlyStream
	IncompatibleStreamHostApi             Error = C.paIncompatibleStreamHostApi
	BadBufferPtr                          Error = C.paBadBufferPtr
	NoDevice                              Error = C.paNoDevice
)

var initialized = 0

// Initialize initializes internal data structures and prepares underlying host APIs for use.
func Initialize() error {
	if initialized <= 0 {
		if err := C.Pa_Initialize(); err != C.paNoError {
			return goError(err)
		}
	}
	initialized++
	return nil
}

// Terminate deallocates all resources allocated by PortAudio since it was initialized by a call to Initialize()
func Terminate() error {
	if initialized > 0 {
		initialized--
		if initialized == 0 {
			if err := C.Pa_Terminate(); err != C.paNoError {
				return goError(err)
			}
		}
	}
	return nil
}

func duration(paTime C.PaTime) time.Duration {
	return time.Duration(paTime * C.PaTime(time.Second))
}

func goError(err C.PaError) error {
	switch err {
	case C.paUnanticipatedHostError:
		return LastHostError()
	case C.paNoError:
		return nil
	}
	return Error(err)
}
