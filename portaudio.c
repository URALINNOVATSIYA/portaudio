#include "_cgo_export.h"

int cb(const void *inputBuffer, void *outputBuffer, unsigned long frames, const PaStreamCallbackTimeInfo *timeInfo, PaStreamCallbackFlags statusFlags, void *userData) {
	return streamCallback((void*)inputBuffer, outputBuffer, frames, (PaStreamCallbackTimeInfo*)timeInfo, statusFlags, userData);
}

//using a variable ensures that the callback signature is checked
PaStreamCallback* paStreamCallback = cb;
PaStreamFinishedCallback* paStreamFinishedCallback = streamFinishedCallback;