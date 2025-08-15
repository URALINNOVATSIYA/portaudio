package portaudio

/*
#cgo pkg-config: portaudio-2.0
#include <portaudio.h>
*/
import "C"

type VersionInfo struct {
	VersionMajor           int
	VersionMinor           int
	VersionSubMinor        int
	VersionControlRevision string
	VersionText            string
}

// Version returns version information for the currently running PortAudio build.
func Version() *VersionInfo {
	info := C.Pa_GetVersionInfo()
	return &VersionInfo{
		VersionMajor:           int(info.versionMajor),
		VersionMinor:           int(info.versionMinor),
		VersionSubMinor:        int(info.versionSubMinor),
		VersionControlRevision: C.GoString(info.versionControlRevision),
		VersionText:            C.GoString(info.versionText),
	}
}

// VersionText returns the textual description of the PortAudio release.
func VersionText() string {
	return C.GoString(C.Pa_GetVersionText())
}

// VersionNumber returns the release number of the currently running PortAudio build.
func VersionNumber() int {
	return int(C.Pa_GetVersion())
}
