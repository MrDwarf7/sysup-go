// Package program loads user execution specs from TOML.
package program

// Spec is one user program loaded from TOML.
type Spec struct {
	Name        string   `toml:"name"`
	Alias       string   `toml:"alias"`
	Description string   `toml:"description"`
	Optional    bool     `toml:"optional"`
	Parallel    bool     `toml:"parallel"`
	Command     []string `toml:"command"`
	// Source is the file the spec was loaded from. It is not a TOML key.
	Source string `toml:"-"`
}
