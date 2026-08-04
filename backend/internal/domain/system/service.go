package system

import "runtime"

type Service struct {
	version string
}

func NewService(version string) *Service {
	return &Service{version: version}
}

func (s *Service) Health() HealthResponse {
	return HealthResponse{Status: "ok"}
}

func (s *Service) Version() VersionResponse {
	return VersionResponse{Version: s.version, GoVersion: runtime.Version()}
}
