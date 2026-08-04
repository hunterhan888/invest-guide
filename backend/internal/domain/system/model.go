package system

type HealthResponse struct {
	Status string `json:"status"`
}

type VersionResponse struct {
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
}
