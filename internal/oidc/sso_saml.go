package oidc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	dsig "github.com/russellhaering/goxmldsig"

	"vard/internal/jose"
	"vard/internal/model"
	"vard/internal/store"
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
	key, _, err := s.sealer.Open(connection.SPKeyWhere(), connection.SPKey)
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

// completeSAML checks a held SAMLResponse (base64, as posted) and reads who it
// signs in. crewjam checks the signature, issuer, audience, destination,
// recipient, the time windows, and that the signed assertion answers the
// request this sign-in sent; the rest is here.
func (s *Service) completeSAML(ctx context.Context, connection *model.SSOConnection, login *model.SSOLogin, posted string) (*ssoPerson, error) {
	if posted == "" {
		return nil, upstream("response", errors.New("no SAMLResponse: only the HTTP-POST binding is taken"))
	}
	raw, err := base64.StdEncoding.DecodeString(posted)
	if err != nil {
		return nil, upstream("response", fmt.Errorf("the SAMLResponse is not base64: %w", err))
	}
	if err := checkSAMLAlgorithms(raw); err != nil {
		return nil, upstream("response", err)
	}

	provider, err := s.samlProvider(connection)
	if err != nil {
		return nil, upstream("metadata", err)
	}

	assertion, err := provider.ParseXMLResponse(raw, []string{login.SAMLRequestID}, provider.AcsURL)
	if err != nil {
		if invalid, ok := errors.AsType[*saml.InvalidResponseError](err); ok {
			err = fmt.Errorf("%w: %v", err, invalid.PrivateErr)
		}
		return nil, upstream("response", err)
	}

	if !bearerConfirmed(assertion) {
		return nil, upstream("response", errors.New("the assertion has no bearer subject confirmation"))
	}

	if err := s.useSAMLAssertion(ctx, connection, assertion); err != nil {
		return nil, err
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

const samlBearer = "urn:oasis:names:tc:SAML:2.0:cm:bearer"

// bearerConfirmed reports whether the subject is confirmed as the Web Browser
// SSO profile requires: by bearer. A holder-of-key or sender-vouches
// confirmation is for another profile, and proves nothing about this browser.
func bearerConfirmed(assertion *saml.Assertion) bool {
	return assertion.Subject != nil && slices.ContainsFunc(assertion.Subject.SubjectConfirmations,
		func(c saml.SubjectConfirmation) bool { return c.Method == samlBearer })
}

// useSAMLAssertion records the signed assertion's ID until it would have
// expired anyway. A sign-in's request is used once and the assertion must
// answer it, so a replay already fails; this holds even if that ever changes.
func (s *Service) useSAMLAssertion(ctx context.Context, connection *model.SSOConnection, assertion *saml.Assertion) error {
	if assertion.ID == "" {
		return upstream("response", errors.New("the assertion has no ID"))
	}

	expires := s.now().Add(saml.MaxIssueDelay)
	if assertion.Conditions != nil && assertion.Conditions.NotOnOrAfter.After(expires) {
		expires = assertion.Conditions.NotOnOrAfter
	}
	for _, confirmation := range assertion.Subject.SubjectConfirmations {
		if data := confirmation.SubjectConfirmationData; data != nil && data.NotOnOrAfter.After(expires) {
			expires = data.NotOnOrAfter
		}
	}

	sum := sha256.Sum256([]byte(connection.ID.String() + "\x00" + assertion.ID))
	err := s.store.UseSAMLAssertion(ctx, hex.EncodeToString(sum[:]), expires.Add(saml.MaxClockSkew))
	switch {
	case errors.Is(err, store.ErrDuplicate):
		return upstream("response", fmt.Errorf("assertion %s was already used", assertion.ID))
	case err != nil:
		return err
	}

	return nil
}

// checkSAMLAlgorithms refuses what a response may name that is broken: SHA-1
// in a signature (collisions are practical), and RSA PKCS#1 v1.5 key
// transport for an encrypted assertion (a padding oracle). The XML
// signature library would accept both. OAEP's own digest is not a signature
// and is left alone; a signature inside an encrypted assertion is only seen
// once it is decrypted, and holds to the library's checks.
func checkSAMLAlgorithms(raw []byte) error {
	document := etree.NewDocument()
	if err := document.ReadFromBytes(raw); err != nil {
		return fmt.Errorf("the SAMLResponse is not XML: %w", err)
	}

	for _, element := range document.FindElements("//*") {
		algorithm := element.SelectAttrValue("Algorithm", "")
		switch {
		case element.Tag == "SignatureMethod" && strings.Contains(algorithm, "sha1"),
			element.Tag == "DigestMethod" && element.Parent() != nil && element.Parent().Tag == "Reference" && strings.HasSuffix(algorithm, "#sha1"):
			return fmt.Errorf("signed with SHA-1 (%s); ask the provider for SHA-256", algorithm)
		case element.Tag == "EncryptionMethod" && strings.HasSuffix(algorithm, "#rsa-1_5"):
			return fmt.Errorf("encrypted with RSA PKCS#1 v1.5 (%s); ask the provider for RSA-OAEP", algorithm)
		}
	}

	return nil
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
	key, _, err := s.sealer.Open(connection.SPKeyWhere(), connection.SPKey)
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
