package ebics

import (
	"context"
	"errors"
	"fmt"

	libebics "code.waarp.fr/lib/ebics/ebics"
	libebicscrypto "code.waarp.fr/lib/ebics/ebics/crypto"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"
)

var errServerSignerNotAvailable = errors.New("ebics request signer not available")

var errServerSignerMissingStaticHeader = errors.New("parse EBICS request signer context: missing static header")

type providerRequestSigner struct {
	store *providerStore
}

func newProviderRequestSigner(store *providerStore) *providerRequestSigner {
	return &providerRequestSigner{store: store}
}

func (s *providerRequestSigner) SignXML(_ []byte) ([]byte, error) {
	return nil, errServerSignerNotAvailable
}

func (s *providerRequestSigner) VerifyXML(input []byte) error {
	if s == nil || s.store == nil {
		return errServerSignerNotAvailable
	}

	req, err := ebicsxml.UnmarshalRequestEnvelope(input)
	if err != nil {
		return fmt.Errorf("parse EBICS request envelope for signature verification: %w", err)
	}
	if req.Header == nil || req.Header.Static == nil {
		return errServerSignerMissingStaticHeader
	}

	keys, err := s.store.GetSubscriberKeys(
		context.Background(),
		libebics.HostID(req.Header.Static.HostID),
		libebics.PartnerID(req.Header.Static.PartnerID),
		libebics.UserID(req.Header.Static.UserID),
	)
	if err != nil {
		return fmt.Errorf("resolve EBICS subscriber signer: %w", err)
	}

	certificate := keys.AuthCertificate
	if len(certificate) == 0 {
		certificate = keys.SigCertificate
	}
	if len(certificate) == 0 {
		return fmt.Errorf("%w: host=%q partner=%q user=%q",
			errServerSignerNotAvailable,
			req.Header.Static.HostID,
			req.Header.Static.PartnerID,
			req.Header.Static.UserID,
		)
	}

	certs, err := libebicscrypto.ParseCertificatesPEM(certificate)
	if err != nil {
		return fmt.Errorf("parse subscriber signer certificate: %w", err)
	}
	if len(certs) == 0 {
		return errServerSignerNotAvailable
	}

	verifier := &libebicscrypto.XMLDSigSigner{
		Certificate: certs[0],
	}

	if err = verifier.VerifyXML(input); err != nil {
		return fmt.Errorf("verify EBICS request XML signature: %w", err)
	}

	return nil
}
