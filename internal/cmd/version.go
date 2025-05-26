package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Show detailed version information including build details.`,
	Run: func(cmd *cobra.Command, args []string) {
		showVersion()
	},
}

func showVersion() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		// Fallback to compile-time variables
		fmt.Printf("wtree version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built: %s\n", date)
		fmt.Printf("  go: %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	// Use build info from runtime
	fmt.Printf("wtree version %s\n", getVersion(info))
	
	// Show VCS information if available
	vcsRevision := ""
	vcsTime := ""
	vcsModified := false
	
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			vcsRevision = setting.Value
		case "vcs.time":
			vcsTime = setting.Value
		case "vcs.modified":
			vcsModified = setting.Value == "true"
		}
	}
	
	if vcsRevision != "" {
		fmt.Printf("  commit: %s\n", vcsRevision)
		if vcsModified {
			fmt.Printf("  modified: true\n")
		}
	}
	
	if vcsTime != "" {
		fmt.Printf("  built: %s\n", vcsTime)
	}
	
	fmt.Printf("  go: %s\n", info.GoVersion)
	fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	
	// Show module information
	if info.Main.Path != "" {
		fmt.Printf("  module: %s\n", info.Main.Path)
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			fmt.Printf("  module version: %s\n", info.Main.Version)
		}
	}
}

func getVersion(info *debug.BuildInfo) string {
	// If we have a module version, use it
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	
	// Otherwise fall back to compile-time version
	if version != "dev" {
		return version
	}
	
	// If still no version, try to extract from VCS
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			// Return first 7 characters of commit hash
			if len(setting.Value) > 7 {
				return setting.Value[:7]
			}
			return setting.Value
		}
	}
	
	return "dev"
}