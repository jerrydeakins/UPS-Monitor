package tray

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"

	xdraw "golang.org/x/image/draw"

	"ups-monitor/assets"
)

type iconState int

const (
	iconOnline iconState = iota
	iconOnBatt
	iconDisconnected
	iconPaused
)

func makeIcon(state iconState) []byte {
	var data []byte

	switch state {
	case iconOnline:
		data = assets.TrayOnlinePNG

	case iconOnBatt:
		data = assets.TrayOnBattPNG

	case iconDisconnected:
		data = assets.TrayDisconnectedPNG

	case iconPaused:
		data = assets.TrayPausedPNG

	default:
		data = assets.TrayOnlinePNG
	}

	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	return createICO(src)
}

func createICO(src image.Image) []byte {
	const (
		size16 = 16
		size20 = 20
		size24 = 24
		size32 = 32
	)

	sizes := []int{
		size16,
		size20,
		size24,
		size32,
	}

	images := make([][]byte, 0, len(sizes))

	for _, size := range sizes {
		img := resizeIcon(src, size)

		var pngData bytes.Buffer

		if err := png.Encode(&pngData, img); err != nil {
			return nil
		}

		images = append(images, pngData.Bytes())
	}

	return buildICO(images, sizes)
}

func resizeIcon(src image.Image, size int) *image.RGBA {
	src = trimTransparent(src)

	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Оставляем небольшой отступ от краёв.
	maxHeight := int(float64(size) * 0.94)
	maxWidth := int(float64(size) * 0.90)

	scale := float64(maxHeight) / float64(srcHeight)

	width := int(float64(srcWidth) * scale * 1.18)
	height := maxHeight

	if width > maxWidth {
		width = maxWidth
		height = int(float64(srcHeight) * float64(width) / float64(srcWidth))
	}

	resized := image.NewRGBA(
		image.Rect(0, 0, width, height),
	)

	xdraw.CatmullRom.Scale(
		resized,
		resized.Bounds(),
		src,
		src.Bounds(),
		xdraw.Over,
		nil,
	)

	// Размещаем по центру квадрата.
	dst := image.NewRGBA(
		image.Rect(0, 0, size, size),
	)

	offsetX := (size - width) / 2
	offsetY := (size - height) / 2

	xdraw.Draw(
		dst,
		image.Rect(
			offsetX,
			offsetY,
			offsetX+width,
			offsetY+height,
		),
		resized,
		image.Point{},
		xdraw.Over,
	)

	return dst
}

func buildICO(images [][]byte, sizes []int) []byte {
	const headerSize = 6
	const directoryEntrySize = 16

	dataOffset := headerSize + directoryEntrySize*len(images)

	var ico bytes.Buffer

	// ICO header.
	binary.Write(
		&ico,
		binary.LittleEndian,
		uint16(0),
	)

	// Image type: ICO.
	binary.Write(
		&ico,
		binary.LittleEndian,
		uint16(1),
	)

	// Number of images.
	binary.Write(
		&ico,
		binary.LittleEndian,
		uint16(len(images)),
	)

	offset := dataOffset

	for i, data := range images {
		size := sizes[i]

		// Width.
		ico.WriteByte(byte(size))

		// Height.
		ico.WriteByte(byte(size))

		// Color palette.
		ico.WriteByte(0)

		// Reserved.
		ico.WriteByte(0)

		// Color planes.
		binary.Write(
			&ico,
			binary.LittleEndian,
			uint16(1),
		)

		// Bits per pixel.
		binary.Write(
			&ico,
			binary.LittleEndian,
			uint16(32),
		)

		// PNG data size.
		binary.Write(
			&ico,
			binary.LittleEndian,
			uint32(len(data)),
		)

		// PNG data offset.
		binary.Write(
			&ico,
			binary.LittleEndian,
			uint32(offset),
		)

		offset += len(data)
	}

	// PNG images.
	for _, data := range images {
		ico.Write(data)
	}

	return ico.Bytes()
}

func trimTransparent(src image.Image) image.Image {
	bounds := src.Bounds()

	minX := bounds.Max.X
	minY := bounds.Max.Y
	maxX := bounds.Min.X
	maxY := bounds.Min.Y

	found := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()

			if a == 0 {
				continue
			}

			found = true

			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}

	if !found {
		return src
	}

	dst := image.NewRGBA(
		image.Rect(0, 0, maxX-minX+1, maxY-minY+1),
	)

	xdraw.Draw(
		dst,
		dst.Bounds(),
		src,
		image.Point{X: minX, Y: minY},
		xdraw.Src,
	)

	return dst
}
