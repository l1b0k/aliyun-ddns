package version

import (
	"fmt"
)

var (
	gitVer    string
	version   string
	buildTime string
)

func Print() string {
	return fmt.Sprintf("%s-%s %s", version, gitVer, buildTime)
}
