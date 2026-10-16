package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// pngHeaderOnly returns a PNG signature + IHDR chunk declaring w x h pixels.
// It is a few dozen bytes but would need GBs of RAM to decode.
func pngHeaderOnly(w, h uint32) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6 // 8-bit RGBA
	_ = binary.Write(&buf, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	buf.Write(chunk)
	_ = binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return buf.Bytes()
}

func TestUploadProfilePicture_RejectsDecompressionBomb(t *testing.T) {
	svc := &ProfilePictureService{}
	data := pngHeaderOnly(50000, 50000)
	_, err := svc.UploadProfilePicture(context.Background(), uuid.New(), bytes.NewReader(data), int64(len(data)), "image/png")
	assert.ErrorIs(t, err, ErrImageTooLarge)
}
