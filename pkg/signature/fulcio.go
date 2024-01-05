package signature

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/sigstore/fulcio/pkg/certificate"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// FulcioIdentity identifies a Fulcio certificate.
type FulcioIdentity struct {
	Issuer         string `yaml:"issuer"`
	SubjectAltName string `yaml:"subject-alt-name"`

	// Reference https://github.com/sigstore/fulcio/blob/df01ed80f075e585dbd617d175fb029c2b9f8165/docs/oid-info.md
	GitHubWorkflowTrigger    string `yaml:"github-workflow-trigger"`
	GitHubWorkflowSha        string `yaml:"github-workflow-sha"`
	GitHubWorkflowName       string `yaml:"github-workflow-name"`
	GithubWorkflowRepository string `yaml:"github-workflow-repository"`
	GitHubWorkflowRef        string `yaml:"github-workflow-ref"`

	BuildSignerURI                      string `yaml:"build-signer-uri"`
	BuildSignerDigest                   string `yaml:"build-signer-digest"`
	RunnerEnvironment                   string `yaml:"runner-environment"`
	SourceRepositoryURI                 string `yaml:"source-repository-uri"`
	SourceRepositoryDigest              string `yaml:"source-repository-digest"`
	SourceRepositoryRef                 string `yaml:"source-repository-ref"`
	SourceRepositoryIdentifier          string `yaml:"source-repository-identifier"`
	SourceRepositoryOwnerURI            string `yaml:"source-repository-owner-uri"`
	SourceRepositoryOwnerIdentifier     string `yaml:"source-repository-owner-identifier"`
	BuildConfigURI                      string `yaml:"build-config-uri"`
	BuildConfigDigest                   string `yaml:"build-config-digest"`
	BuildTrigger                        string `yaml:"build-trigger"`
	RunInvocationURI                    string `yaml:"run-invocation-uri"`
	SourceRepositoryVisibilityAtSigning string `yaml:"source-repository-visibility-at-signing"`

	BuildTriggerPattern string `yaml:"build-trigger-pattern"`
}

func (i FulcioIdentity) values() map[string]string {
	ret := map[string]string{}
	if i.Issuer != "" {
		ret[certificate.OIDIssuer.String()] = i.Issuer //nolint:staticcheck
	} else {
		ret[certificate.OIDIssuer.String()] = "https://token.actions.githubusercontent.com" //nolint:staticcheck
	}
	if i.SubjectAltName != "" {
		ret[cryptoutils.SANOID.String()] = i.SubjectAltName
	}

	if i.GitHubWorkflowTrigger != "" {
		ret[certificate.OIDGitHubWorkflowTrigger.String()] = i.GitHubWorkflowTrigger //nolint:staticcheck
	}
	if i.GitHubWorkflowSha != "" {
		ret[certificate.OIDGitHubWorkflowSHA.String()] = i.GitHubWorkflowSha //nolint:staticcheck
	}
	if i.GitHubWorkflowName != "" {
		ret[certificate.OIDGitHubWorkflowName.String()] = i.GitHubWorkflowName //nolint:staticcheck
	}
	if i.GithubWorkflowRepository != "" {
		ret[certificate.OIDGitHubWorkflowRepository.String()] = i.GithubWorkflowRepository //nolint:staticcheck
	}
	if i.GitHubWorkflowRef != "" {
		ret[certificate.OIDGitHubWorkflowRef.String()] = i.GitHubWorkflowRef //nolint:staticcheck
	}

	if i.BuildSignerURI != "" {
		ret[certificate.OIDBuildSignerURI.String()] = i.BuildSignerURI
	}
	if i.BuildSignerDigest != "" {
		ret[certificate.OIDBuildSignerDigest.String()] = i.BuildSignerDigest
	}
	if i.RunnerEnvironment != "" {
		ret[certificate.OIDRunnerEnvironment.String()] = i.RunnerEnvironment
	}
	if i.SourceRepositoryURI != "" {
		ret[certificate.OIDSourceRepositoryURI.String()] = i.SourceRepositoryURI
	}
	if i.SourceRepositoryDigest != "" {
		ret[certificate.OIDSourceRepositoryDigest.String()] = i.SourceRepositoryDigest
	}
	if i.SourceRepositoryRef != "" {
		ret[certificate.OIDSourceRepositoryRef.String()] = i.SourceRepositoryRef
	}
	if i.SourceRepositoryIdentifier != "" {
		ret[certificate.OIDSourceRepositoryIdentifier.String()] = i.SourceRepositoryIdentifier
	}
	if i.SourceRepositoryOwnerURI != "" {
		ret[certificate.OIDSourceRepositoryOwnerURI.String()] = i.SourceRepositoryOwnerURI
	}
	if i.SourceRepositoryOwnerIdentifier != "" {
		ret[certificate.OIDSourceRepositoryOwnerIdentifier.String()] = i.SourceRepositoryOwnerIdentifier
	}
	if i.BuildConfigURI != "" {
		ret[certificate.OIDBuildConfigURI.String()] = i.BuildConfigURI
	}
	if i.BuildConfigDigest != "" {
		ret[certificate.OIDBuildConfigDigest.String()] = i.BuildConfigDigest
	}
	if i.BuildTrigger != "" {
		ret[certificate.OIDBuildTrigger.String()] = i.BuildTrigger
	}
	if i.RunInvocationURI != "" {
		ret[certificate.OIDRunInvocationURI.String()] = i.RunInvocationURI
	}
	if i.SourceRepositoryVisibilityAtSigning != "" {
		ret[certificate.OIDSourceRepositoryVisibilityAtSigning.String()] = i.SourceRepositoryVisibilityAtSigning
	}

	return ret
}
func (i FulcioIdentity) regexps() (map[string]*regexp.Regexp, error) {
	ret := map[string]*regexp.Regexp{}

	add := func(pattern string, oid asn1.ObjectIdentifier) error {
		if pattern == "" {
			return nil
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("compiling %s pattern: %w", oid, err)
		}
		ret[oid.String()] = re
		return nil
	}

	if err := add(i.BuildTriggerPattern, certificate.OIDBuildTrigger); err != nil {
		return nil, err
	}

	return ret, nil
}

var (
	oidSubjectKeyIdentifier = asn1.ObjectIdentifier{2, 5, 29, 14}
	oidKeyUsage             = asn1.ObjectIdentifier{2, 5, 29, 15}
	authorityKeyIdentifier  = asn1.ObjectIdentifier{2, 5, 29, 35}
	oidExtendedKeyUsage     = asn1.ObjectIdentifier{2, 5, 29, 37}
)

func decodeExtension(e pkix.Extension) (ret string, err error) {
	switch {
	// BEGIN: Deprecated
	case e.Id.Equal(certificate.OIDIssuer): //nolint:staticcheck
		return string(e.Value), nil
	case e.Id.Equal(certificate.OIDGitHubWorkflowTrigger): //nolint:staticcheck
		return string(e.Value), nil
	case e.Id.Equal(certificate.OIDGitHubWorkflowSHA): //nolint:staticcheck
		return string(e.Value), nil
	case e.Id.Equal(certificate.OIDGitHubWorkflowName): //nolint:staticcheck
		return string(e.Value), nil
	case e.Id.Equal(certificate.OIDGitHubWorkflowRepository): //nolint:staticcheck
		return string(e.Value), nil
	case e.Id.Equal(certificate.OIDGitHubWorkflowRef): //nolint:staticcheck
		return string(e.Value), nil
	// END: Deprecated
	case e.Id.Equal(certificate.OIDIssuerV2):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDBuildSignerURI):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDBuildSignerDigest):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDRunnerEnvironment):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDSourceRepositoryURI):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDSourceRepositoryDigest):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDSourceRepositoryRef):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDSourceRepositoryIdentifier):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDSourceRepositoryOwnerURI):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDSourceRepositoryOwnerIdentifier):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDBuildConfigURI):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDBuildConfigDigest):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDBuildTrigger):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDRunInvocationURI):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(certificate.OIDSourceRepositoryVisibilityAtSigning):
		err = certificate.ParseDERString(e.Value, &ret)
	case e.Id.Equal(cryptoutils.SANOID):
		var seq asn1.RawValue
		rest, err := asn1.Unmarshal(e.Value, &seq)
		if err != nil {
			return "", err
		} else if len(rest) != 0 {
			return "", fmt.Errorf("trailing data after X.509 extension")
		}
		if !seq.IsCompound || seq.Tag != asn1.TagSequence || seq.Class != asn1.ClassUniversal {
			return "", asn1.StructuralError{Msg: "bad SAN sequence"}
		}
		rest = seq.Bytes
		for len(rest) > 0 {
			var v asn1.RawValue
			rest, err = asn1.Unmarshal(rest, &v)
			if err != nil {
				return "", err
			}

			switch v.Tag {
			case 1:
				return string(v.Bytes), nil
			}
		}

	// Ignore these extensions
	case e.Id.Equal(oidSubjectKeyIdentifier):
	case e.Id.Equal(oidKeyUsage):
	case e.Id.Equal(authorityKeyIdentifier):
	case e.Id.Equal(oidExtendedKeyUsage):

	default:
		slog.Warn("unknown extension", slog.String("id", e.Id.String()))
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("decoding extension %s: %w", e.Id, err)
	}
	return ret, nil
}
