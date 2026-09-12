package program

const (
	FileName = "programs.toml"
	DirName  = "programs"
)

type Spec struct {
	Name        string   `toml:"name"`
	Alias       string   `toml:"alias"`
	Description string   `toml:"description"`
	Optional    bool     `toml:"optional"`
	Parallel    bool     `toml:"parallel"`
	Command     []string `toml:"command"`
	Retries     Retries  `toml:"retries"`
	Source      string   `toml:"-"`
}

// Retries is per-recipe. Always retries any failure; otherwise only
// classified errors (paru --noconfirm conflicts, dropped AUR RPC).
// MaxAttempts is total runs including the first. 0 means unset (use
// global always/max). A positive global max always caps this value.
type Retries struct {
	Always      bool `toml:"always"`
	MaxAttempts int  `toml:"max_attempts"`
}
