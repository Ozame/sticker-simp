package imaging

import (
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"

	"github.com/anthonynsimon/bild/transform"

	_ "image/gif"
	_ "image/jpeg"
)

// RecodeAndScale scales the image to fitting size, and recodes it into a png image.
// Supports png, jpeg and gif formats for the original image.
func RecodeAndScale(r io.Reader, w io.Writer) error {
	img, _, err := image.Decode(r)
	if err != nil {
		return err
	}
	resized := scale(img)

	resultImg := &notOpaqueRGBA{image.NewRGBA(resized.Bounds())}
	draw.Draw(resultImg, resized.Bounds(), resized, image.Point{}, draw.Src)

	png.Encode(w, resultImg)
	return nil
}

// scale resizes the given image to fit a 512x512 sized rectangle
func scale(img image.Image) *image.RGBA {
	size := img.Bounds().Size()
	var newX, newY int
	ratio := float64(size.X) / float64(size.Y)
	if size.X <= size.Y {
		newY = 512
		newX = int(math.Floor(ratio * float64(512)))
	} else {
		newX = 512
		newY = int(math.Floor(float64(512) / ratio))
	}
	return transform.Resize(img, newX, newY, transform.Linear)
}

// enforce image.RGBA to always add the alpha channel when encoding PNGs.
type notOpaqueRGBA struct {
	*image.RGBA
}

func (i *notOpaqueRGBA) Opaque() bool {
	return false
}
