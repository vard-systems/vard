package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"vard/internal/database/databasetest"
	"vard/internal/jose"
	"vard/internal/model"
)

// legacySeal is how a value was sealed before versions existed.
func legacySeal(t *testing.T, secret string, plain []byte) []byte {
	t.Helper()
	sum := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, aead.NonceSize())
	_, _ = rand.Read(nonce)
	return aead.Seal(nonce, nonce, plain, nil)
}

// After a rotation and a reseal, everything stored opens with the new key
// alone, where it is kept: values from before versions and values under the
// previous key alike. A second reseal finds nothing to do.
func TestLiveResealingMovesEverySecretToTheCurrentKey(t *testing.T) {
	const before, now = "the-key-before-0123456789abcdefghijkl", "the-key-now-0123456789abcdefghijklmno"
	db, _ := databasetest.Open(t)
	st := New(db)
	ctx := context.Background()

	old, err := jose.NewSealer(before)
	if err != nil {
		t.Fatal(err)
	}

	mail := model.MailSettings{Host: "smtp.example.com", Port: 587, Encryption: model.MailStartTLS}
	mail.Password = legacySeal(t, before, []byte("smtp password"))
	if err := db.Create(&mail).Error; err != nil {
		t.Fatal(err)
	}

	provider := model.SocialProvider{Kind: model.SocialGoogle, Slug: "google", Name: "Google", ClientID: "id", Scopes: model.StringList{}}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatal(err)
	}
	secret, err := old.SealFor(provider.ClientSecretWhere(), []byte("client secret"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&provider).Update("client_secret", secret).Error; err != nil {
		t.Fatal(err)
	}

	rotated, err := jose.NewSealer(now, before)
	if err != nil {
		t.Fatal(err)
	}
	reseal := func(where jose.Context, sealed []byte) ([]byte, bool, error) {
		plain, stale, err := rotated.OpenFor(where, sealed)
		if err != nil || !stale {
			return nil, false, err
		}
		resealed, err := rotated.SealFor(where, plain)
		return resealed, true, err
	}

	changed, err := st.ResealAll(ctx, reseal)
	if err != nil {
		t.Fatal(err)
	}
	if changed != 2 {
		t.Errorf("resealed %d values, want the 2 stored", changed)
	}

	only, err := jose.NewSealer(now)
	if err != nil {
		t.Fatal(err)
	}
	var stored model.MailSettings
	if err := db.First(&stored, "id = ?", mail.ID).Error; err != nil {
		t.Fatal(err)
	}
	if plain, _, err := only.OpenFor(stored.PasswordWhere(), stored.Password); err != nil || string(plain) != "smtp password" {
		t.Errorf("the mail password after reseal = %q, %v", plain, err)
	}
	var kept model.SocialProvider
	if err := db.First(&kept, "id = ?", provider.ID).Error; err != nil {
		t.Fatal(err)
	}
	if plain, _, err := only.OpenFor(kept.ClientSecretWhere(), kept.ClientSecret); err != nil || string(plain) != "client secret" {
		t.Errorf("the client secret after reseal = %q, %v", plain, err)
	}

	if again, err := st.ResealAll(ctx, reseal); err != nil || again != 0 {
		t.Errorf("a second reseal changed %d, %v; want nothing", again, err)
	}
}
