package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_Health(t *testing.T) {
	s := NewService("1.0.0")
	assert.Equal(t, "ok", s.Health().Status)
}

func TestService_Version(t *testing.T) {
	s := NewService("1.0.0")
	v := s.Version()
	assert.Equal(t, "1.0.0", v.Version)
	assert.NotEmpty(t, v.GoVersion)
}
