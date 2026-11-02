package services

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

// T264: thumbnail replaces nfnt/resize.Thumbnail — fit within the box,
// keep the aspect ratio, never upscale.
func TestThumbnail(t *testing.T) {
	cases := []struct {
		w, h, wantW, wantH int
	}{
		{800, 400, 200, 100}, // landscape: width-bound
		{300, 600, 100, 200}, // portrait: height-bound
		{1000, 1000, 200, 200},
		{120, 80, 120, 80}, // already small: unchanged
	}
	for _, c := range cases {
		got := thumbnail(image.NewRGBA(image.Rect(0, 0, c.w, c.h)), TargetWidth, TargetHeight).Bounds()
		assert.Equal(t, c.wantW, got.Dx(), "%dx%d width", c.w, c.h)
		assert.Equal(t, c.wantH, got.Dy(), "%dx%d height", c.w, c.h)
	}
}
