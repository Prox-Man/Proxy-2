package options

// Header represents an individual header that will be added to a request or
// response header.
type Header struct {
	// Name is the header name to be used for this set of values.
	// Names should be unique within a list of Headers.
	Name string `yaml:"name,omitempty"`

	// PreserveRequestValue determines whether any values for this header
	// should be preserved for the request to the upstream server.
	// This option only applies to injected request headers.
	// Defaults to false (headers that match this header will be stripped).
	PreserveRequestValue bool `yaml:"preserveRequestValue,omitempty"`

	// Values contains the desired values for this header
	Values []HeaderValue `yaml:"values,omitempty"`
}

// HeaderValue represents a single header value and the sources that can
// make up the header value
type HeaderValue struct {
	// Allow users to load the value from a secret source
	// TODO(tuunit): Investigate why mapstructure: ",squash" doesn't work with pointers
	SecretSource `yaml:",squash"`

	// Allow users to load the value from a session claim
	// TODO(tuunit): Investigate why mapstructure: ",squash" doesn't work with pointers
	ClaimSource `yaml:",squash"`
}

// ClaimSource allows loading a header value from a claim within the session
type ClaimSource struct {
	// Claim is the name of the claim in the session that the value should be
	// loaded from.
	Claim string `yaml:"claim,omitempty"`

	// Prefix is an optional prefix that will be prepended to the value of the
	// claim if it is non-empty.
	Prefix string `yaml:"prefix,omitempty"`

	// BasicAuthPassword converts this claim into a basic auth header.
	// Note the value of claim will become the basic auth username and the
	// basicAuthPassword will be used as the password value.
	BasicAuthPassword *SecretSource `yaml:"basicAuthPassword,omitempty"`
}

func (s *ClaimSource) IsSet() bool {
	return s.Claim != "" || s.Prefix != "" || s.BasicAuthPassword != nil
}
