package model

import (
	"bytes"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func encoded(t *testing.T, encode func(*bytes.Buffer, image.Image) error, width, height int) []byte {
	t.Helper()

	var out bytes.Buffer
	if err := encode(&out, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}

	return out.Bytes()
}

func pngOf(t *testing.T, width, height int) []byte {
	return encoded(t, func(b *bytes.Buffer, m image.Image) error { return png.Encode(b, m) }, width, height)
}

func TestNewOrganizationLogo(t *testing.T) {
	logo := pngOf(t, 128, 128)

	tests := []struct {
		name string
		data []byte
		want error
		typ  string
	}{
		{name: "a PNG", data: logo, typ: "image/png"},
		{
			name: "a JPEG",
			data: encoded(t, func(b *bytes.Buffer, m image.Image) error { return jpeg.Encode(b, m, nil) }, 64, 64),
			typ:  "image/jpeg",
		},
		// An SVG served from the apps' own origin could run script.
		{name: "an SVG", data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), want: ErrLogoType},
		{
			name: "a GIF",
			data: encoded(t, func(b *bytes.Buffer, m image.Image) error { return gif.Encode(b, m, nil) }, 64, 64),
			want: ErrLogoType,
		},
		{name: "a page", data: []byte("<!doctype html><title>logo.png</title>"), want: ErrLogoType},
		{name: "a PNG's signature and nothing after it", data: logo[:12], want: ErrLogoType},
		{name: "more than 512 KB", data: append(append([]byte{}, logo...), make([]byte, LogoMaxBytes)...), want: ErrLogoTooLarge},
		{name: "too small to see", data: pngOf(t, 8, 64), want: ErrLogoDimension},
		{name: "wider than any header", data: pngOf(t, 5000, 16), want: ErrLogoDimension},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logo, err := NewOrganizationLogo(tt.data)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if err != nil {
				return
			}

			if logo.ContentType != tt.typ || len(logo.Version) != 16 {
				t.Errorf("logo = %s, version %q; want %s and 16 characters", logo.ContentType, logo.Version, tt.typ)
			}
		})
	}
}

// The version is what a browser keeps the logo by, so a new logo must have a
// new one and the same logo the same one.
func TestOrganizationLogoVersion(t *testing.T) {
	first, _ := NewOrganizationLogo(pngOf(t, 64, 64))
	again, _ := NewOrganizationLogo(pngOf(t, 64, 64))
	other, _ := NewOrganizationLogo(pngOf(t, 64, 65))

	if first.Version != again.Version || first.Version == other.Version {
		t.Errorf("versions %q, %q, %q: want the same logo alike and another different", first.Version, again.Version, other.Version)
	}
}
