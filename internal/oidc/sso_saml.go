package oidc

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	dsig "github.com/russellhaering/goxmldsig"

	"loginer/internal/jose"
	"loginer/internal/model"
)

func ParseSAMLMetadata(metadata []byte) (*saml.EntityDescriptor, error) {
	descriptor, err := samlsp.ParseMetadata(metadata)
	if err != nil {
		return nil, fmt.Errorf("the metadata is not SAML metadata: %w", err)
	}

	if len(descriptor.IDPSSODescriptors) == 0 {
		return nil, fmt.Errorf("the metadata describes no identity provider")
	}

	idp := descriptor.IDPSSODescriptors[0]

	redirect := false
	for _, endpoint := range idp.SingleSignOnServices {
		if endpoint.Binding == saml.HTTPRedirectBinding {
			redirect = true
		}
	}
	if !redirect {
		return nil, fmt.Errorf("the identity provider has no HTTP-Redirect sign-in address")
	}

	signing := false
	for _, key := range idp.KeyDescriptors {
		if (key.Use == "" || key.Use == "signing") && len(key.KeyInfo.X509Data.X509Certificates) > 0 {
			signing = true
		}
	}
	if !signing {
		return nil, fmt.Errorf("the metadata has no signing certificate")
	}

	return descriptor, nil
}

func (s *Service) FetchSAMLMetadata(ctx context.Context, address string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, socialTimeout)
	defer cancel()

	response, err := s.social.Do(request.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s answered %d", request.URL.Host, response.StatusCode)
	}
	if _, err := ParseSAMLMetadata(body); err != nil {
		return "", err
	}

	return string(body), nil
}

func (s *Service) samlProvider(connection *model.SSOConnection) (*saml.ServiceProvider, error) {
	key, err := s.sealer.Open(connection.SPKey)
	if err != nil {
		return nil, fmt.Errorf("open the connection's key: %w", err)
	}

	certificate, err := jose.ParseCertificate(connection.SPCertificate)
	if err != nil {
		return nil, err
	}

	idp, err := ParseSAMLMetadata([]byte(connection.Metadata))
	if err != nil {
		return nil, err
	}

	acs, err := url.Parse(connection.ACSURL(s.issuer))
	if err != nil {
		return nil, err
	}
	metadata, err := url.Parse(connection.MetadataURLFor(s.issuer))
	if err != nil {
		return nil, err
	}

	provider := &saml.ServiceProvider{
		EntityID:          connection.EntityID(s.issuer),
		Key:               key,
		Certificate:       certificate,
		AcsURL:            *acs,
		MetadataURL:       *metadata,
		IDPMetadata:       idp,
		AuthnNameIDFormat: saml.NameIDFormat(connection.NameIDFormat.URN()),
		AllowIDPInitiated: false,
	}
	if connection.SignRequests {
		provider.SignatureMethod = dsig.RSASHA256SignatureMethod
	}

	return provider, nil
}

func (s *Service) startSAML(connection *model.SSOConnection, login *model.SSOLogin, state string) (string, error) {
	provider, err := s.samlProvider(connection)
	if err != nil {
		return "", upstream("metadata", err)
	}

	request, err := provider.MakeAuthenticationRequest(
		provider.GetSSOBindingLocation(saml.HTTPRedirectBinding), saml.HTTPRedirectBinding, saml.HTTPPostBinding,
	)
	if err != nil {
		return "", err
	}

	location, err := request.Redirect(state, provider)
	if err != nil {
		return "", err
	}

	login.SAMLRequestID = request.ID

	return location.String(), nil
}

func (s *Service) completeSAML(connection *model.SSOConnection, login *model.SSOLogin, r *http.Request) (*ssoPerson, error) {
	provider, err := s.samlProvider(connection)
	if err != nil {
		return nil, upstream("metadata", err)
	}

	assertion, err := provider.ParseResponse(r, []string{login.SAMLRequestID})
	if err != nil {
		if invalid, ok := errors.AsType[*saml.InvalidResponseError](err); ok {
			err = fmt.Errorf("%w: %v", err, invalid.PrivateErr)
		}
		return nil, upstream("response", err)
	}

	attributes := map[string][]string{}
	for _, statement := range assertion.AttributeStatements {
		for _, attribute := range statement.Attributes {
			var values []string
			for _, value := range attribute.Values {
				if text := strings.TrimSpace(value.Value); text != "" {
					values = append(values, text)
				}
			}
			attributes[attribute.Name] = values
			if attribute.FriendlyName != "" {
				attributes[attribute.FriendlyName] = values
			}
		}
	}

	first := func(names []string) string {
		for _, name := range names {
			if values := attributes[name]; len(values) > 0 {
				return values[0]
			}
		}
		return ""
	}

	person := &ssoPerson{
		Email:     first(connection.Attribute("email")),
		FirstName: first(connection.Attribute("first_name")),
		LastName:  first(connection.Attribute("last_name")),
		Sent:      slices.Sorted(maps.Keys(attributes)),
	}
	for _, name := range connection.Attribute("groups") {
		if values := attributes[name]; len(values) > 0 {
			person.Groups = values
			break
		}
	}

	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		person.Subject = assertion.Subject.NameID.Value

		if person.Email == "" && strings.Contains(person.Subject, "@") {
			person.Email = person.Subject
		}
	}

	return person, nil
}

func (s *Service) SSOMetadata(ctx context.Context, slug string) ([]byte, error) {
	connection, err := s.store.SSOConnectionBySlug(ctx, slug)
	if err != nil || connection.Protocol != model.SSOProtocolSAML {
		return nil, ErrSSOUnknown
	}

	provider, err := s.samlProvider(connection)
	if err != nil {
		provider, err = s.bareSAMLProvider(connection)
		if err != nil {
			return nil, err
		}
	}

	document := provider.Metadata()
	document.SPSSODescriptors[0].AuthnRequestsSigned = &connection.SignRequests
	wantSigned := true
	document.SPSSODescriptors[0].WantAssertionsSigned = &wantSigned

	return xml.MarshalIndent(document, "", "  ")
}

func (s *Service) bareSAMLProvider(connection *model.SSOConnection) (*saml.ServiceProvider, error) {
	key, err := s.sealer.Open(connection.SPKey)
	if err != nil {
		return nil, err
	}
	certificate, err := jose.ParseCertificate(connection.SPCertificate)
	if err != nil {
		return nil, err
	}
	acs, _ := url.Parse(connection.ACSURL(s.issuer))
	metadata, _ := url.Parse(connection.MetadataURLFor(s.issuer))

	return &saml.ServiceProvider{
		EntityID:          connection.EntityID(s.issuer),
		Key:               key,
		Certificate:       certificate,
		AcsURL:            *acs,
		MetadataURL:       *metadata,
		AuthnNameIDFormat: saml.NameIDFormat(connection.NameIDFormat.URN()),
	}, nil
}
