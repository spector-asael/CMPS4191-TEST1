package main

import (
	"image"
	"image/color"
	"testing"
)

func TestCalculateFit(t *testing.T) {
	tests := []struct {
		name         string
		srcW, srcH   int
		maxW, maxH   int
		wantW, wantH int
	}{
		{
			name:         "3:2 landscape downscale to 800x600 (spec example)",
			srcW:         3000,
			srcH:         2000,
			maxW:         800,
			maxH:         600,
			wantW:        800,
			wantH:        533,
		},
		{
			name:         "3:2 landscape downscale to 1200x900 (spec example)",
			srcW:         3000,
			srcH:         2000,
			maxW:         1200,
			maxH:         900,
			wantW:        1200,
			wantH:        800,
		},
		{
			name:         "16:9 landscape downscale to 800x600",
			srcW:         1920,
			srcH:         1080,
			maxW:         800,
			maxH:         600,
			wantW:        800,
			wantH:        450,
		},
		{
			name:         "9:16 portrait downscale to 800x600",
			srcW:         1080,
			srcH:         1920,
			maxW:         800,
			maxH:         600,
			wantW:        338,
			wantH:        600,
		},
		{
			name:         "Smaller than max bounds keeps original size without upscaling",
			srcW:         400,
			srcH:         300,
			maxW:         800,
			maxH:         600,
			wantW:        400,
			wantH:        300,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotW, gotH := calculateFit(tc.srcW, tc.srcH, tc.maxW, tc.maxH)
			if gotW != tc.wantW || gotH != tc.wantH {
				t.Errorf("calculateFit(%d, %d, %d, %d) = (%d, %d); want (%d, %d)",
					tc.srcW, tc.srcH, tc.maxW, tc.maxH, gotW, gotH, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestCropCenterSquare(t *testing.T) {
	// Create a 200x100 rectangle image
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	cropped := cropCenterSquare(img)
	bounds := cropped.Bounds()

	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("cropCenterSquare bounds = %dx%d; want 100x100", bounds.Dx(), bounds.Dy())
	}

	// Now test square thumbnail resizing
	thumb := resizeImage(cropped, 150, 150)
	thumbBounds := thumb.Bounds()
	if thumbBounds.Dx() != 150 || thumbBounds.Dy() != 150 {
		t.Errorf("thumbnail bounds = %dx%d; want 150x150", thumbBounds.Dx(), thumbBounds.Dy())
	}
}
