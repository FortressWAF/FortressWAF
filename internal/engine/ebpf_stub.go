//go:build !linux

package engine

import "log/slog"

type EBPFTelemetry struct {
	devMode bool
}

func NewEBPFTelemetry(devMode bool, iface string, port, sampleRate int) *EBPFTelemetry {
	if devMode {
		slog.Debug("ebpf: telemetry not available on this platform")
	}
	return &EBPFTelemetry{devMode: devMode}
}

func (e *EBPFTelemetry) Name() string { return "ebpf" }

func (e *EBPFTelemetry) Inspect(ctx *RequestContext) (*Decision, error) {
	return nil, nil
}

func (e *EBPFTelemetry) Start() {}

func (e *EBPFTelemetry) Stop() {}

func (e *EBPFTelemetry) Stats() map[string]uint64 {
	return nil
}
