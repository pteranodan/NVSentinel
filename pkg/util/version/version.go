package version

import (
	"fmt"
	"runtime"

	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/component-base/compatibility"
)

var (
	GitVersion = "v0.0.0-devel"
	GitCommit  = "unknown"
	BuildDate  = "unknown"
)

type Info struct {
	GitVersion string
	GitCommit  string
	BuildDate  string
	GoVersion  string
	Compiler   string
	Platform   string
}

func Get() Info {
	return Info{
		GitVersion: GitVersion,
		GitCommit:  GitCommit,
		BuildDate:  BuildDate,
		GoVersion:  runtime.Version(),
		Compiler:   runtime.Compiler,
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func (i Info) String() string {
	return i.GitVersion
}

// UserAgent returns the standard user agent string for clients.
func UserAgent() string {
	return fmt.Sprintf("nvidia-device-api/%s (%s)", GitVersion, Get().Platform)
}

func RegisterComponent(registry compatibility.ComponentGlobalsRegistry) {
	v, err := utilversion.ParseSemantic(GitVersion)
	if err != nil {
		v = utilversion.MustParseSemantic("v0.0.1")
	}

	binaryVersion := v
	emulationVersion := v
	minCompatibilityVersion := v

	effectiveVer := compatibility.NewEffectiveVersion(
		binaryVersion,
		false,
		emulationVersion,
		minCompatibilityVersion,
	)

	registry.Register("nvidia-device-api", effectiveVer, nil)
}
