package resolve

type Helper string

const (
	HelperParu Helper = "paru"
	HelperYay  Helper = "yay"

	EnvPkgManager = "PKG_MANAGER"
)

var FallbackHelpers = []Helper{
	HelperParu,
	HelperYay,
}

func (h Helper) String() string {
	return string(h)
}
