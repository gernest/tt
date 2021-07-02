package middlewares

import "context"

// Matcher reports whether hostname matches the Matcher's criteria.
type Matcher func(ctx context.Context, hostname string) bool

// Equals is a trivial Matcher that implements string equality.
func Equals(want string) Matcher {
	return func(_ context.Context, got string) bool {
		return want == got
	}
}
