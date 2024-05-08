package signature

import (
	"crypto/x509"
	"log/slog"
	"regexp"
	"strings"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

type CertificateVerifier struct {
	values  map[string]string
	regexps map[string]*regexp.Regexp
}

func NewCertificateVerifier(identity FulcioIdentity) (*CertificateVerifier, error) {
	regexps, err := identity.regexps()
	if err != nil {
		return nil, err
	}
	return &CertificateVerifier{
		values:  identity.values(),
		regexps: regexps,
	}, nil
}

func (v CertificateVerifier) Verify(version string, cert *x509.Certificate) (bool, error) {
	vCount := len(v.values)
	reCount := len(v.regexps)
	slog.Debug("verifying extensions", slog.Int("needed_values", vCount), slog.Int("needed_regexps", reCount), slog.Int("extension_count", len(cert.Extensions)))

	if expected, ok := v.values[cryptoutils.SANOID.String()]; ok {
		var matched bool
		for _, uri := range cert.URIs {
			if uri.String() == expected {
				matched = true
				break
			}
		}

		if !matched {
			slog.Debug("subject alt name mismatch", slog.String("expected", expected))
			return false, nil
		}
		vCount--
	}

	for _, e := range cert.Extensions {
		actual, err := decodeExtension(e)
		if err != nil {
			return false, err
		} else if actual == "" {
			continue
		}
		extID := e.Id.String()
		log := slog.With(slog.String("id", extID), slog.String("actual", actual))
		if expected, ok := v.values[extID]; ok {
			if actual != strings.ReplaceAll(expected, "{{VERSION}}", version) {
				log.Debug("extension value mismatch", slog.String("expected", expected))
				return false, nil
			}
			log.Debug("extension value match", slog.String("expected", expected))
			vCount--
		}

		if re, ok := v.regexps[extID]; ok {
			if !re.MatchString(actual) {
				log.Debug("extension regexp mismatch", slog.String("expected", re.String()))
				return false, nil
			}
			log.Debug("extension regexp match", slog.String("expected", re.String()))
			reCount--
		}

		if vCount == 0 && reCount == 0 {
			return true, nil
		}
	}
	return false, nil
}
