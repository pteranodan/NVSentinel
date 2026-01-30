package verflag

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/nvidia/nvsentinel/pkg/util/version"
	"github.com/spf13/pflag"
)

type versionValue int

const (
	VersionFalse versionValue = iota
	VersionTrue
	VersionRaw
)

const versionFlagName = "version"

var (
	versionFlag = Version(versionFlagName, VersionFalse, "--version, --version=raw prints version information and quits; --version=vX.Y.Z... sets the reported version")
	programName = "NVIDIA Device API"
)

// variables for unit testing PrintAndExitIfRequested
var (
	output = io.Writer(os.Stdout)
	exit   = os.Exit
)

// Version defines a flag with the specified name, default value, and usage string.
func Version(name string, value versionValue, usage string) *versionValue {
	p := new(versionValue)
	*p = value
	pflag.Var(p, name, usage)
	pflag.Lookup(name).NoOptDefVal = "true"
	return p
}

func (v *versionValue) IsBoolFlag() bool { return true }
func (v *versionValue) Get() interface{} { return versionValue(*v) }

func (v *versionValue) Set(s string) error {
	if s == "raw" {
		*v = VersionRaw
		return nil
	}
	boolVal, err := strconv.ParseBool(s)
	if err == nil {
		if boolVal {
			*v = VersionTrue
		} else {
			*v = VersionFalse
		}
		return nil
	}
	*v = VersionTrue
	return nil
}

func (v *versionValue) String() string {
	return fmt.Sprintf("%v", *v)
}

func (v *versionValue) Type() string {
	return "version"
}

// PrintAndExitIfRequested checks if the version flag was passed and prints accordingly.
func PrintAndExitIfRequested() {
	if *versionFlag == VersionRaw {
		fmt.Fprintf(output, "%#v\n", version.Get())
		exit(0)
	} else if *versionFlag == VersionTrue {
		printVersionTable()
		exit(0)
	}
}

func printVersionTable() {
	v := version.Get()
	w := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "%s\n", programName)
	fmt.Fprintf(w, "---\t---\n")
	fmt.Fprintf(w, "Version\t%s\n", v.GitVersion)
	fmt.Fprintf(w, "GitCommit\t%s\n", v.GitCommit)
	fmt.Fprintf(w, "BuildDate\t%s\n", v.BuildDate)
	fmt.Fprintf(w, "GoVersion\t%s\n", v.GoVersion)
	fmt.Fprintf(w, "Platform\t%s\n", v.Platform)

	w.Flush()
}

// AddFlags registers the version flag on a specific FlagSet.
func AddFlags(fs *pflag.FlagSet) {
	fs.AddFlag(pflag.Lookup(versionFlagName))
}
