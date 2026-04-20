// Package fixture is a tiny test module for golden-testing the indexer.
package fixture

// Greeter produces greetings.
type Greeter struct {
	Prefix string
}

// Hello returns a greeting for name.
func (g *Greeter) Hello(name string) string {
	return g.Prefix + " " + name
}

// Anon is an anonymous helper function.
func Anon() int { return 42 }

// MaxRetries bounds retry attempts.
const MaxRetries = 3
