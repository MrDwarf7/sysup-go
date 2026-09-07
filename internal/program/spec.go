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
	Source      string   `toml:"-"`
}
