package services

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // register the PNG decoder
	"net/http"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register the WebP decoder
)

// Job photo processing (T174). Every photo is decoded and re-encoded as
// JPEG: that strips EXIF (GPS, device serials) and anything smuggled after
// the image data. The EXIF orientation is applied to the pixels first, so a
// phone photo taken sideways still shows upright.

const (
	// AttachmentMaxSide is the longest side of the stored photo.
	AttachmentMaxSide = 2048
	// AttachmentThumbSide is the longest side of a photo's thumbnail.
	AttachmentThumbSide = 400
	// MaxAttachmentDimension bounds each declared side of an uploaded photo,
	// and MaxAttachmentPixels its area, checked with image.DecodeConfig
	// before anything is decoded (security audit F-19: a few-KB PNG can
	// declare 50k x 50k pixels). 40 MP covers phone cameras' default output.
	MaxAttachmentDimension = 12000
	MaxAttachmentPixels    = 40_000_000

	attachmentJPEGQuality = 82
	thumbJPEGQuality      = 75
)

// Sniffed content types accepted for attachments: the client's declared
// type is ignored.
const (
	contentTypeJPEG = "image/jpeg"
	contentTypePNG  = "image/png"
	contentTypeWebP = "image/webp"
	contentTypePDF  = "application/pdf"
)

// sniffAttachment returns the content type of data from its bytes, or ""
// when it isn't an allowed type.
func sniffAttachment(data []byte) string {
	switch ct := http.DetectContentType(data); ct {
	case contentTypeJPEG, contentTypePNG, contentTypeWebP, contentTypePDF:
		return ct
	}
	return ""
}

// processedPhoto is a photo ready to store: the re-encoded JPEG, its
// thumbnail and the stored (oriented, scaled) dimensions.
type processedPhoto struct {
	Full          []byte
	Thumb         []byte
	Width, Height int
}

// processPhoto checks the declared dimensions, decodes, orients, scales
// and re-encodes data.
func processPhoto(data []byte) (*processedPhoto, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrAttachmentUnreadable
	}
	if cfg.Width <= 0 || cfg.Height <= 0 ||
		cfg.Width > MaxAttachmentDimension || cfg.Height > MaxAttachmentDimension ||
		int64(cfg.Width)*int64(cfg.Height) > MaxAttachmentPixels {
		return nil, ErrAttachmentDimensions
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrAttachmentUnreadable
	}
	orientation := 1
	if format == "jpeg" {
		orientation = jpegOrientation(data)
	}
	full := applyOrientation(fitWithin(img, AttachmentMaxSide), orientation)
	thumb := fitWithin(full, AttachmentThumbSide)

	var fullBuf, thumbBuf bytes.Buffer
	if err := jpeg.Encode(&fullBuf, full, &jpeg.Options{Quality: attachmentJPEGQuality}); err != nil {
		return nil, err
	}
	if err := jpeg.Encode(&thumbBuf, thumb, &jpeg.Options{Quality: thumbJPEGQuality}); err != nil {
		return nil, err
	}
	b := full.Bounds()
	return &processedPhoto{Full: fullBuf.Bytes(), Thumb: thumbBuf.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// fitWithin returns img scaled down (never up) so its longest side is at
// most maxSide, as a fresh RGBA on white (JPEG has no transparency).
func fitWithin(img image.Image, maxSide int) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxSide || h > maxSide {
		if w >= h {
			w, h = maxSide, max(1, h*maxSide/b.Dx())
		} else {
			w, h = max(1, w*maxSide/b.Dy()), maxSide
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	if w == b.Dx() && h == b.Dy() {
		draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Over)
	} else {
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	}
	return dst
}

// applyOrientation turns src as EXIF orientation o (1-8) says, so the pixels
// display upright without the tag.
func applyOrientation(src *image.RGBA, o int) *image.RGBA {
	if o < 2 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2: // mirrored
				nx, ny = w-1-x, y
			case 3: // rotated 180
				nx, ny = w-1-x, h-1-y
			case 4: // mirrored vertically
				nx, ny = x, h-1-y
			case 5: // transposed
				nx, ny = y, x
			case 6: // needs 90 degrees clockwise
				nx, ny = h-1-y, x
			case 7: // transversed
				nx, ny = h-1-y, w-1-x
			case 8: // needs 90 degrees counter-clockwise
				nx, ny = y, w-1-x
			}
			si := src.PixOffset(b.Min.X+x, b.Min.Y+y)
			di := dst.PixOffset(nx, ny)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}

// jpegOrientation reads the EXIF orientation tag (0x0112) from a JPEG's APP1
// segment; 1 (upright) when absent or unreadable.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		switch {
		case marker == 0xFF: // fill byte
			i++
			continue
		case marker == 0x01 || (marker >= 0xD0 && marker <= 0xD8): // no length
			i += 2
			continue
		case marker == 0xDA || marker == 0xD9: // image data starts: no EXIF
			return 1
		}
		segLen := int(binary.BigEndian.Uint16(data[i+2:]))
		if segLen < 2 || i+2+segLen > len(data) {
			return 1
		}
		seg := data[i+4 : i+2+segLen]
		if marker == 0xE1 && len(seg) >= 6 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + segLen
	}
	return 1
}

// tiffOrientation finds the orientation tag in IFD0 of a TIFF block.
func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	if bo.Uint16(t[2:]) != 42 {
		return 1
	}
	off := int64(bo.Uint32(t[4:]))
	if off < 8 || off+2 > int64(len(t)) {
		return 1
	}
	n := int64(bo.Uint16(t[off:]))
	for k := int64(0); k < n; k++ {
		e := off + 2 + k*12
		if e+12 > int64(len(t)) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 {
			if v := int(bo.Uint16(t[e+8:])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}
