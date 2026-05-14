package cards

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestRenderHalfBlockImageUsesTransparentLetterboxing(t *testing.T) {
	t.Parallel()
	img := image.NewRGBA(image.Rect(0, 0, 4, 20))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetRGBA(x, y, color.RGBA{R: 220, G: 40, B: 30, A: 255})
		}
	}

	output := renderHalfBlockImage(img, 30, 6)
	if strings.Contains(output, "0;0;0") {
		t.Fatalf("rendered image contains black letterbox color escapes:\n%s", output)
	}
	if !strings.Contains(output, "\x1b[0m ") {
		t.Fatalf("rendered image should use reset spaces for transparent letterbox cells:\n%s", output)
	}
}

func TestFitRectPreservesSourceAspect(t *testing.T) {
	t.Parallel()
	src := image.Rect(10, 20, 110, 220)
	dst := image.Rect(0, 0, 80, 40)

	fitted := fitRect(dst, src)
	if fitted.Dx() != 20 || fitted.Dy() != 40 {
		t.Fatalf("fitted rect = %v, want 20x40", fitted)
	}
	if fitted.Min.X != 30 || fitted.Max.X != 50 {
		t.Fatalf("fitted rect should be centered horizontally, got %v", fitted)
	}
}
