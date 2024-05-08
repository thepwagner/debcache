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
	// Issuer is the issuer of the certificate. Defaults to "https://token.actions.githubusercontent.com".
	Issuer         string `yaml:"issuer"`
	SubjectAltName string `yaml:"subject-alt-name"`

	// Reference https://github.com/sigstore/fulcio/blob/df01ed80f075e585dbd617d175fb029c2b9f8165/docs/oid-info.md
	GitHubWorkflowTrigger    string `yaml:"github-workflow-trigger"`
	GitHubWorkflowSha        string `yaml:"github-workflow-sha"`
	GitHubWorkflowName       string `yaml:"github-workflow-name"`
	GitHubWorkflowRepository string `yaml:"github-workflow-repository"`
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

	IssuerPattern         string `yaml:"issuer-pattern"`
	SubjectAltNamePattern string `yaml:"subject-alt-name-pattern"`

	GitHubWorkflowTriggerPattern    string `yaml:"github-workflow-trigger-pattern"`
	GitHubWorkflowShaPattern        string `yaml:"github-workflow-sha-pattern"`
	GitHubWorkflowNamePattern       string `yaml:"github-workflow-name-pattern"`
	GitHubWorkflowRepositoryPattern string `yaml:"github-workflow-repository-pattern"`
	GitHubWorkflowRefPattern        string `yaml:"github-workflow-ref-pattern"`

	BuildSignerURIPattern                      string `yaml:"build-signer-uri-pattern"`
	BuildSignerDigestPattern                   string `yaml:"build-signer-digest-pattern"`
	RunnerEnvironmentPattern                   string `yaml:"runner-environment-pattern"`
	SourceRepositoryURIPattern                 string `yaml:"source-repository-uri-pattern"`
	SourceRepositoryDigestPattern              string `yaml:"source-repository-digest-pattern"`
	SourceRepositoryRefPattern                 string `yaml:"source-repository-ref-pattern"`
	SourceRepositoryIdentifierPattern          string `yaml:"source-repository-identifier-pattern"`
	SourceRepositoryOwnerURIPattern            string `yaml:"source-repository-owner-uri-pattern"`
	SourceRepositoryOwnerIdentifierPattern     string `yaml:"source-repository-owner-identifier-pattern"`
	BuildConfigURIPattern                      string `yaml:"build-config-uri-pattern"`
	BuildConfigDigestPattern                   string `yaml:"build-config-digest-pattern"`
	BuildTriggerPattern                        string `yaml:"build-trigger-pattern"`
	RunInvocationURIPattern                    string `yaml:"run-invocation-uri-pattern"`
	SourceRepositoryVisibilityAtSigningPattern string `yaml:"source-repository-visibility-at-signing-pattern"`
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
	if i.GitHubWorkflowRepository != "" {
		ret[certificate.OIDGitHubWorkflowRepository.String()] = i.GitHubWorkflowRepository //nolint:staticcheck
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

//nolint:staticcheck
func (i FulcioIdentity) regexps() (map[string]*regexp.Regexp, error) {
	ret := map[string]*regexp.Regexp{}

	for oid, pattern := range map[string]string{
		certificate.OIDIssuer.String(): i.IssuerPattern,
		cryptoutils.SANOID.String():    i.SubjectAltNamePattern,

		certificate.OIDGitHubWorkflowTrigger.String():    i.GitHubWorkflowTriggerPattern,
		certificate.OIDGitHubWorkflowSHA.String():        i.GitHubWorkflowShaPattern,
		certificate.OIDGitHubWorkflowName.String():       i.GitHubWorkflowNamePattern,
		certificate.OIDGitHubWorkflowRepository.String(): i.GitHubWorkflowRepositoryPattern,
		certificate.OIDGitHubWorkflowRef.String():        i.GitHubWorkflowRefPattern,

		certificate.OIDBuildSignerURI.String():                      i.BuildSignerURIPattern,
		certificate.OIDBuildSignerDigest.String():                   i.BuildSignerDigestPattern,
		certificate.OIDRunnerEnvironment.String():                   i.RunnerEnvironmentPattern,
		certificate.OIDSourceRepositoryURI.String():                 i.SourceRepositoryURIPattern,
		certificate.OIDSourceRepositoryDigest.String():              i.SourceRepositoryDigestPattern,
		certificate.OIDSourceRepositoryRef.String():                 i.SourceRepositoryRefPattern,
		certificate.OIDSourceRepositoryIdentifier.String():          i.SourceRepositoryIdentifierPattern,
		certificate.OIDSourceRepositoryOwnerURI.String():            i.SourceRepositoryOwnerURIPattern,
		certificate.OIDSourceRepositoryOwnerIdentifier.String():     i.SourceRepositoryOwnerIdentifierPattern,
		certificate.OIDBuildConfigURI.String():                      i.BuildConfigURIPattern,
		certificate.OIDBuildConfigDigest.String():                   i.BuildConfigDigestPattern,
		certificate.OIDBuildTrigger.String():                        i.BuildTriggerPattern,
		certificate.OIDRunInvocationURI.String():                    i.RunInvocationURIPattern,
		certificate.OIDSourceRepositoryVisibilityAtSigning.String(): i.SourceRepositoryVisibilityAtSigningPattern,
	} {
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling %s pattern: %w", oid, err)
		}
		ret[oid] = re
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
