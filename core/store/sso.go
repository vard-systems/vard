package store

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/cache"
)

func (s *Store) SSOConnections(ctx context.Context) ([]model.SSOConnection, error) {
	var connections []model.SSOConnection
	err := s.db.WithContext(ctx).Order("name").Find(&connections).Error

	return connections, err
}

func (s *Store) SSOConnection(ctx context.Context, id uuid.UUID) (*model.SSOConnection, error) {
	var connection model.SSOConnection
	if err := s.db.WithContext(ctx).First(&connection, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &connection, nil
}

func (s *Store) SSOConnectionBySlug(ctx context.Context, slug string) (*model.SSOConnection, error) {
	var connection model.SSOConnection
	if err := s.db.WithContext(ctx).First(&connection, "slug = ?", slug).Error; err != nil {
		return nil, translate(err)
	}

	return &connection, nil
}

func (s *Store) SSOConnectionForEmail(ctx context.Context, email string) (*model.SSOConnection, error) {
	_, domain, ok := strings.Cut(strings.ToLower(strings.TrimSpace(email)), "@")
	if !ok || domain == "" {
		return nil, ErrNotFound
	}

	var connection model.SSOConnection
	err := s.db.WithContext(ctx).
		Where("is_enabled = ? AND jsonb_exists(domains, ?)", true, domain).
		First(&connection).Error
	if err != nil {
		return nil, translate(err)
	}

	return &connection, nil
}

func (s *Store) SSOButtons(ctx context.Context) ([]model.SSOConnection, error) {
	return cached(ctx, s, cache.SSOButtons, "buttons", func() ([]model.SSOConnection, error) {
		var connections []model.SSOConnection
		err := s.db.WithContext(ctx).
			Where("is_enabled = ? AND show_on_login = ?", true, true).
			Order("name").
			Find(&connections).Error

		return connections, err
	})
}

func (s *Store) AnySSOConnectionEnabled(ctx context.Context) (bool, error) {
	return cached(ctx, s, cache.SSOButtons, "available", func() (bool, error) {
		var count int64
		err := s.db.WithContext(ctx).Model(&model.SSOConnection{}).
			Where("is_enabled = ? AND jsonb_array_length(domains) > 0", true).
			Limit(1).Count(&count).Error

		return count > 0, err
	})
}

func (s *Store) SSODomainsTaken(ctx context.Context, domains []string, except uuid.UUID) ([]string, error) {
	connections, err := s.SSOConnections(ctx)
	if err != nil {
		return nil, err
	}

	var taken []string
	for _, connection := range connections {
		if connection.ID == except {
			continue
		}
		for _, domain := range domains {
			if slices.Contains(connection.Domains, domain) && !slices.Contains(taken, domain) {
				taken = append(taken, domain)
			}
		}
	}

	return taken, nil
}

func (s *Store) CreateSSOConnection(ctx context.Context, connection *model.SSOConnection) error {
	return s.forgetting(ctx, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := translate(tx.Create(connection).Error); err != nil {
			return err
		}

		return challengeDomainsIn(tx, connection)
	}), cache.SSOButtons)
}

// challengeDomainsIn keeps a connection's proofs in step with its domains: a
// new domain gets a challenge to prove, a removed one loses its proof, so
// adding it back is proved again.
func challengeDomainsIn(tx *gorm.DB, connection *model.SSOConnection) error {
	err := tx.Where("connection_id = ? AND domain NOT IN ?", connection.ID, append([]string{""}, connection.Domains...)).
		Delete(&model.SSODomainVerification{}).Error
	if err != nil {
		return err
	}

	for _, domain := range connection.Domains {
		_, token, err := model.NewSecret()
		if err != nil {
			return err
		}

		challenge := model.SSODomainVerification{ConnectionID: connection.ID, Domain: domain, Token: token}
		err = tx.Omit(clause.Associations).
			Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "connection_id"}, {Name: "domain"}}, DoNothing: true}).
			Create(&challenge).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// SSODomainVerifications are a connection's domains and whether each is
// proved, in the order the connection lists them.
func (s *Store) SSODomainVerifications(ctx context.Context, connection *model.SSOConnection) ([]model.SSODomainVerification, error) {
	var found []model.SSODomainVerification
	if err := s.db.WithContext(ctx).Where("connection_id = ?", connection.ID).Find(&found).Error; err != nil {
		return nil, err
	}

	out := make([]model.SSODomainVerification, 0, len(found))
	for _, domain := range connection.Domains {
		for _, verification := range found {
			if verification.Domain == domain {
				out = append(out, verification)
			}
		}
	}

	return out, nil
}

// SSODomainVerification is one of a connection's domains with its proof.
func (s *Store) SSODomainVerification(ctx context.Context, connection uuid.UUID, domain string) (*model.SSODomainVerification, error) {
	var verification model.SSODomainVerification
	err := s.db.WithContext(ctx).First(&verification, "connection_id = ? AND domain = ?", connection, domain).Error
	if err != nil {
		return nil, translate(err)
	}

	return &verification, nil
}

// CheckedSSODomain records a DNS check of a domain, and its proof when the
// check found the record. A domain already proved stays proved.
func (s *Store) CheckedSSODomain(ctx context.Context, verification *model.SSODomainVerification, proved bool, at time.Time) error {
	upgrade := proved && !verification.Proved()

	changes := map[string]any{"checked_at": at}
	if upgrade {
		changes["verified_at"] = at
		changes["method"] = model.SSODomainProvedByDNS
	}

	err := s.db.WithContext(ctx).Model(&model.SSODomainVerification{}).Where("id = ?", verification.ID).Updates(changes).Error
	if err != nil {
		return err
	}

	verification.CheckedAt = &at
	if upgrade {
		verification.VerifiedAt = &at
		verification.Method = model.SSODomainProvedByDNS
	}

	return nil
}

// SSODomainProved says whether a connection has proved it holds a domain.
func (s *Store) SSODomainProved(ctx context.Context, connection uuid.UUID, domain string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.SSODomainVerification{}).
		Where("connection_id = ? AND domain = ? AND verified_at IS NOT NULL", connection, domain).
		Count(&count).Error

	return count > 0, err
}

// SaveSSOConnection writes a connection back; with `newProvider` it also drops
// its identities, since a subject is only unique within the provider that
// issued it. Accounts stay and re-link by address.
func (s *Store) SaveSSOConnection(ctx context.Context, connection *model.SSOConnection, newProvider bool) error {
	return s.forgetting(ctx, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if newProvider {
			err := tx.Where("connection_id = ?", connection.ID).Delete(&model.SSOIdentity{}).Error
			if err != nil {
				return translate(err)
			}
		}

		if err := translate(tx.Save(connection).Error); err != nil {
			return err
		}

		return challengeDomainsIn(tx, connection)
	}), cache.SSOButtons)
}

func (s *Store) DeleteSSOConnection(ctx context.Context, connection *model.SSOConnection) error {
	return s.forgetting(ctx, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, child := range []any{&model.SSOIdentity{}, &model.SSOLogin{}} {
			if err := tx.Where("connection_id = ?", connection.ID).Delete(child).Error; err != nil {
				return translate(err)
			}
		}

		return translate(tx.Delete(connection).Error)
	}), cache.SSOButtons)
}

func (s *Store) SSOIdentityCounts(ctx context.Context) (map[uuid.UUID]int64, error) {
	var rows []struct {
		ConnectionID uuid.UUID
		Count        int64
	}

	err := s.db.WithContext(ctx).
		Model(&model.SSOIdentity{}).
		Select("connection_id, COUNT(*) AS count").
		Group("connection_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		out[row.ConnectionID] = row.Count
	}

	return out, nil
}

func (s *Store) SSOIdentity(ctx context.Context, connection uuid.UUID, subject string) (*model.SSOIdentity, error) {
	var identity model.SSOIdentity
	err := s.db.WithContext(ctx).First(&identity, "connection_id = ? AND subject = ?", connection, subject).Error
	if err != nil {
		return nil, translate(err)
	}

	return &identity, nil
}

func (s *Store) CreateSSOIdentity(ctx context.Context, identity *model.SSOIdentity) error {
	return translate(s.db.WithContext(ctx).Create(identity).Error)
}

func (s *Store) MarkSSOIdentityUsed(ctx context.Context, identity *model.SSOIdentity, email string, at time.Time) error {
	return translate(s.db.WithContext(ctx).
		Model(identity).
		Updates(map[string]any{"last_login_at": at, "email": email}).Error)
}

func (s *Store) CreateSSOLogin(ctx context.Context, login *model.SSOLogin) error {
	return translate(s.db.WithContext(ctx).Create(login).Error)
}

func (s *Store) TakeSSOLogin(ctx context.Context, stateHash string, now time.Time) (*model.SSOLogin, error) {
	var login model.SSOLogin

	err := s.db.WithContext(ctx).
		Clauses(clause.Returning{}).
		Where("state_hash = ? AND expires_at > ?", stateHash, now).
		Delete(&login).Error
	if err != nil {
		return nil, translate(err)
	}

	if login.ID == uuid.Nil {
		return nil, ErrNotFound
	}

	return &login, nil
}

// HoldSAMLResponse keeps a provider's answer on the sign-in its relay handle
// names, once: a second answer to the same sign-in is refused.
func (s *Store) HoldSAMLResponse(ctx context.Context, connectionID uuid.UUID, relayHash, sealed string, now time.Time) error {
	result := s.db.WithContext(ctx).
		Model(&model.SSOLogin{}).
		Where("relay_hash = ? AND relay_hash <> '' AND connection_id = ? AND expires_at > ? AND saml_response = ''",
			relayHash, connectionID, now).
		Update("saml_response", sealed)
	if result.Error != nil {
		return translate(result.Error)
	}
	if result.RowsAffected != 1 {
		return ErrNotFound
	}

	return nil
}

// TakeHeldSAMLLogin uses up the sign-in a state belongs to, only once its
// provider's answer is held: a browser sent here early leaves its sign-in
// alone.
func (s *Store) TakeHeldSAMLLogin(ctx context.Context, stateHash string, now time.Time) (*model.SSOLogin, error) {
	var login model.SSOLogin

	err := s.db.WithContext(ctx).
		Clauses(clause.Returning{}).
		Where("state_hash = ? AND expires_at > ? AND saml_response <> ''", stateHash, now).
		Delete(&login).Error
	if err != nil {
		return nil, translate(err)
	}

	if login.ID == uuid.Nil {
		return nil, ErrNotFound
	}

	return &login, nil
}

// UseSAMLAssertion records an assertion as used, or answers ErrDuplicate if it
// already was.
func (s *Store) UseSAMLAssertion(ctx context.Context, hash string, expires time.Time) error {
	return translate(s.db.WithContext(ctx).Create(&model.SAMLAssertion{Hash: hash, ExpiresAt: expires}).Error)
}
