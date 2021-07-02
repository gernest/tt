package tcp

// Config contains the proxying state for one listener.
type Config struct {
	Routes      []Route
	AcmeTargets []Target // accumulates targets that should be probed for acme.
	AllowACME   bool     // if true, AddSNIRoute doesn't add targets to acmeTargets.
	Network     string
}
