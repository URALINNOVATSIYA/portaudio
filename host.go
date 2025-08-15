package portaudio

/*
#cgo pkg-config: portaudio-2.0
#include <portaudio.h>
*/
import "C"
import "fmt"

type HostErrorInfo struct {
	HostApiType HostApiType
	Code        int
	Text        string
}

func (err HostErrorInfo) Error() string {
	return fmt.Sprintf("%s; code: %d; api type: %s", err.Text, err.Code, err.HostApiType)
}

// PortAudio Api types.
const (
	InDevelopment   HostApiType = C.paInDevelopment
	DirectSound     HostApiType = C.paDirectSound
	MME             HostApiType = C.paMME
	ASIO            HostApiType = C.paASIO
	SoundManager    HostApiType = C.paSoundManager
	CoreAudio       HostApiType = C.paCoreAudio
	OSS             HostApiType = C.paOSS
	ALSA            HostApiType = C.paALSA
	AL              HostApiType = C.paAL
	BeOS            HostApiType = C.paBeOS
	WDMkS           HostApiType = C.paWDMKS
	JACK            HostApiType = C.paJACK
	WASAPI          HostApiType = C.paWASAPI
	AudioScienceHPI HostApiType = C.paAudioScienceHPI
)

type HostApiType int

func (t HostApiType) String() string {
	return hostApiStrings[t]
}

var hostApiStrings = [...]string{
	InDevelopment:   "InDevelopment",
	DirectSound:     "DirectSound",
	MME:             "MME",
	ASIO:            "ASIO",
	SoundManager:    "SoundManager",
	CoreAudio:       "CoreAudio",
	OSS:             "OSS",
	ALSA:            "ALSA",
	AL:              "AL",
	BeOS:            "BeOS",
	WDMkS:           "WDMKS",
	JACK:            "JACK",
	WASAPI:          "WASAPI",
	AudioScienceHPI: "AudioScienceHPI",
}

type HostApiInfo struct {
	Type                HostApiType
	Name                string
	DefaultInputDevice  *DeviceInfo
	DefaultOutputDevice *DeviceInfo
	Devices             []*DeviceInfo
}

// HostApi returns a pointer to a structure containing information about a specific host Api.
// The returning value is a non-negative value indicating the number of available host APIs.
func HostApi(index int) *HostApiInfo {
	info := C.Pa_GetHostApiInfo(C.PaHostApiIndex(index))
	return &HostApiInfo{
		Type: HostApiType(info._type),
		Name: C.GoString(info.name),
	}
}

// GetHostApiCount returns the number of available host APIs.
// Even if a host API is available it may have no devices available.
func HostApiCount() int {
	return int(C.Pa_GetHostApiCount())
}

// DefaultHostApiIndex returns the index of the default host API.
// The default host API will be the lowest common denominator host API
// on the current platform and is unlikely to provide the best performance.
// The returning value is a non-negative value ranging from 0 to (GetHostApiCount()-1)
func DefaultHostApiIndex() int {
	return int(C.Pa_GetDefaultHostApi())
}

// DefaultHostApi returns information about default host Api.
func DefaultHostApi() *HostApiInfo {
	index := C.Pa_GetDefaultHostApi()
	if index < 0 {
		return nil
	}
	return HostApi(int(index))
}

// LastHostError returns information about the last host error encountered.
func LastHostError() HostErrorInfo {
	info := C.Pa_GetLastHostErrorInfo()
	return HostErrorInfo{
		HostApiType(info.hostApiType),
		int(info.errorCode),
		C.GoString(info.errorText),
	}
}
