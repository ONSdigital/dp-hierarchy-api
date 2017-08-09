package config

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetRetrunsDefaultValues(t *testing.T) {
	t.Parallel()
	Convey("When a loading a configuration, default values are return", t, func() {
		config, err := Get()
		So(err, ShouldBeNil)
		So(config.BindAddr, ShouldEqual, ":22600")
	})
}
