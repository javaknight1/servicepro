package version

import "testing"

func TestGet_FromEnv(t *testing.T) {
	t.Setenv("APP_VERSION", "1.2.3")
	t.Setenv("GIT_SHA", "abc1234")

	got := Get()
	if got.AppVersion != "1.2.3" || got.GitSHA != "abc1234" {
		t.Fatalf("Get() = %+v, want app_version=1.2.3 git_sha=abc1234", got)
	}
}

func TestGet_Fallbacks(t *testing.T) {
	t.Setenv("APP_VERSION", "")
	t.Setenv("GIT_SHA", "")

	got := Get()
	if got.AppVersion != DefaultAppVersion || got.GitSHA != DefaultGitSHA {
		t.Fatalf("Get() = %+v, want fallbacks %q/%q", got, DefaultAppVersion, DefaultGitSHA)
	}
}
