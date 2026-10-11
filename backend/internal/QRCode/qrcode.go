package qrcode

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

func GenerateQRCode(url string) ([]byte, error) {
	qrCode, err := qr.Encode(url, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("failed to encode QR code: %w", err)
	}

	scaledQR, err := barcode.Scale(qrCode, 300, 300)
	if err != nil {
		return nil, fmt.Errorf("failed to scale QR code: %w", err)
	}

	var buf bytes.Buffer
	err = png.Encode(&buf, scaledQR)
	if err != nil {
		return nil, fmt.Errorf("failed to encode QR code image: %w", err)
	}

	return buf.Bytes(), nil
}
