package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"slices"
)

// LogoMaxBytes keeps an upload, base64 in a JSON body, under the API's body
// limit.
const LogoMaxBytes = 512 << 10

const (
	logoMinSide = 16
	logoMaxSide = 4096
)

var (
	ErrLogoTooLarge  = errors.New("the logo is larger than 512 KB")
	ErrLogoType      = errors.New("the logo is not a PNG or JPEG")
	ErrLogoDimension = errors.New("the logo is too small or too large")
)

// logoTypes are the formats accepted. No SVG: it can carry script, and a
// picture served from the apps' own origin would run it.
var logoTypes = []string{"image/png", "image/jpeg"}

// OrganizationLogo is the organisation's app logo, a square mark the panel's
// header and the sign-in pages show, kept in the database like everything
// else. The organisation's row holds its version, so the cached organisation
// never carries the bytes.
type OrganizationLogo struct {
	Base

	ContentType string `gorm:"size:32;not null;default:''" json:"content_type"`

	// Data is only ever sent as the picture itself, never inside JSON.
	Data []byte `gorm:"not null" json:"-"`

	// Version is a digest of Data, put in the logo's address so a browser can
	// keep it for as long as it likes and still sees a new one at once.
	Version string `gorm:"size:16;not null;default:''" json:"version"`
}

func (OrganizationLogo) TableName() string {
	return "organization_logos"
}

// NewOrganizationLogo checks an upload by what it is, never by what it is
// called or claims to be: the bytes have to be a PNG or JPEG whose header
// decodes, of a sensible size. Only the header is decoded, so a small file
// claiming an enormous picture costs nothing.
func NewOrganizationLogo(data []byte) (OrganizationLogo, error) {
	if len(data) > LogoMaxBytes {
		return OrganizationLogo{}, ErrLogoTooLarge
	}

	contentType := http.DetectContentType(data)
	if !slices.Contains(logoTypes, contentType) {
		return OrganizationLogo{}, ErrLogoType
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return OrganizationLogo{}, ErrLogoType
	}

	for _, side := range []int{config.Width, config.Height} {
		if side < logoMinSide || side > logoMaxSide {
			return OrganizationLogo{}, ErrLogoDimension
		}
	}

	digest := sha256.Sum256(data)

	return OrganizationLogo{
		ContentType: contentType,
		Data:        data,
		Version:     hex.EncodeToString(digest[:8]),
	}, nil
}
