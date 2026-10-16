// Package version reports the identity of the running build.
//
// Values come from the environment so a single image can be traced back to
// its commit: the Dockerfile turns the GIT_SHA / APP_VERSION build args into
// ENV in the final stage, and the release pipeline passes them. Local runs
// fall back to "dev" / "unknown".
package version

import "os"

const (
	// DefaultAppVersion is reported when APP_VERSION is unset.
	DefaultAppVersion = "dev"
	// DefaultGitSHA is reported when GIT_SHA is unset.
	DefaultGitSHA = "unknown"
)

// AppVersion returns the release version (APP_VERSION), e.g. "0.8.0".
func AppVersion() string {
	return getenv("APP_VERSION", DefaultAppVersion)
}

// GitSHA returns the commit the build was cut from (GIT_SHA).
func GitSHA() string {
	return getenv("GIT_SHA", DefaultGitSHA)
}

// Info is the JSON body served by GET /version.
type Info struct {
	GitSHA     string `json:"git_sha" example:"3df4280"`
	AppVersion string `json:"app_version" example:"0.8.0"`
} // @name VersionInfo

// Get returns the current build identity.
func Get() Info {
	return Info{GitSHA: GitSHA(), AppVersion: AppVersion()}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
