package signature

import (
	"crypto/x509/pkix"
	"log/slog"
	"regexp"
	"strings"
)

type idVerifier struct {
	values  map[string]string
	regexps map[string]*regexp.Regexp
}

func newIdVerifier(identity FulcioIdentity) (*idVerifier, error) {
	regexps, err := identity.regexps()
	if err != nil {
		return nil, err
	}
	return &idVerifier{
		values:  identity.values(),
		regexps: regexps,
	}, nil
}

func (v idVerifier) verifyExtensions(version string, ext []pkix.Extension) (bool, error) {
	vCount := len(v.values)
	reCount := len(v.regexps)
	slog.Debug("verifying extensions", slog.Int("needed_values", vCount), slog.Int("needed_regexps", reCount), slog.Int("extension_count", len(ext)))
	for _, e := range ext {
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
