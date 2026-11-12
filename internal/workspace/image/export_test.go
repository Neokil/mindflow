package image

import "image"

// OrientForTest exposes the private image transform to external package tests.
func OrientForTest(source image.Image, orientation int) image.Image {
	return orient(source, orientation)
}

// JPEGOrientationForTest exposes EXIF decoding without expanding the production API.
func JPEGOrientationForTest(data []byte) int { return jpegOrientation(data) }
