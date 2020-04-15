package config

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetReturnsDefaultValues(t *testing.T) {
	t.Parallel()
	Convey("When a loading a configuration, default values are return", t, func() {
		config, err := Get()
		So(err, ShouldBeNil)
		So(config, ShouldResemble, &Config{
			BindAddr:                   ":22600",
			HierarchyAPIURL:            "http://localhost:22600",
			CodelistAPIURL:             "http://localhost:22400",
			ShutdownTimeout:            5 * time.Second,
			HealthCheckInterval:        30 * time.Second,
			HealthCheckCriticalTimeout: 90 * time.Second,
		})
	})
}
