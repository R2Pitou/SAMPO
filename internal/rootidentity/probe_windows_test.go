//go:build windows

package rootidentity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"sampo/internal/domain"
)

func TestWindowsNormalExtendedAndVolumeGUIDPathsIdentifyOneRoot(t *testing.T) {
	root := t.TempDir()
	prober := SystemProber{}
	normal := probeStrong(t, prober, root)
	extended := probeStrong(t, prober, `\\?\`+root)
	assertSameRoot(t, normal, extended)

	if !strings.HasPrefix(strings.ToUpper(normal.FinalPathEvidence), `\\?\VOLUME{`) {
		t.Skipf("volume GUID path unavailable: %q", normal.FinalPathEvidence)
	}
	guid := probeStrong(t, prober, normal.FinalPathEvidence)
	assertSameRoot(t, normal, guid)
}

func TestWindowsCaseAndShortNameAliasesIdentifyOneRoot(t *testing.T) {
	longRoot := filepath.Join(t.TempDir(), "Long Provider Root Name")
	if err := os.Mkdir(longRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	prober := SystemProber{}
	normal := probeStrong(t, prober, longRoot)
	caseAlias := probeStrong(t, prober, strings.ToUpper(longRoot))
	assertSameRoot(t, normal, caseAlias)

	short, err := shortPath(longRoot)
	if err != nil || strings.EqualFold(short, longRoot) {
		t.Skipf("8.3 alias unavailable: path=%q err=%v", short, err)
	}
	shortAlias := probeStrong(t, prober, short)
	assertSameRoot(t, normal, shortAlias)
}

func TestWindowsJunctionAndSymlinkTargetsIdentifyOneRoot(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	prober := SystemProber{}
	want := probeStrong(t, prober, target)

	t.Run("junction", func(t *testing.T) {
		link := filepath.Join(parent, "junction")
		if output, err := exec.Command("cmd.exe", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			t.Skipf("junction creation unavailable: %v: %s", err, output)
		}
		got := probeStrong(t, prober, link)
		assertSameRoot(t, want, got)
	})

	t.Run("symlink", func(t *testing.T) {
		link := filepath.Join(parent, "symlink")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symbolic-link privilege unavailable: %v", err)
		}
		got := probeStrong(t, prober, link)
		assertSameRoot(t, want, got)
	})
}

func TestWindowsUNCAndExtendedUNCAliasesIdentifyOneRoot(t *testing.T) {
	root := os.Getenv("SAMPO_TEST_UNC_ROOT")
	if root == "" {
		t.Skip("set SAMPO_TEST_UNC_ROOT to exercise a live UNC provider")
	}
	if !strings.HasPrefix(root, `\\`) || strings.HasPrefix(root, `\\?\`) {
		t.Fatalf("SAMPO_TEST_UNC_ROOT must be an ordinary UNC path, got %q", root)
	}
	extended := `\\?\UNC\` + strings.TrimPrefix(root, `\\`)
	prober := SystemProber{}
	normal, err := prober.Probe(root)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := prober.Probe(extended)
	if err != nil {
		t.Fatal(err)
	}
	if normal.IdentityConfidence != domain.RootIdentityWeak || !normal.CatalogueOnly {
		t.Fatalf("UNC root was not weak catalogue-only: %#v", normal)
	}
	assertSameRoot(t, normal, alias)
}

func TestWindowsProbeRejectsRelativeRoot(t *testing.T) {
	if _, err := (SystemProber{}).Probe("."); err == nil {
		t.Fatal("relative provider root was accepted")
	}
}

func probeStrong(t *testing.T, prober SystemProber, path string) domain.ProviderRoot {
	t.Helper()
	root, err := prober.Probe(path)
	if err != nil {
		t.Fatal(err)
	}
	if root.IdentityConfidence != domain.RootIdentityStrong || root.PhysicalIdentity == "" || root.FallbackIdentity == "" {
		t.Fatalf("root identity is not strong: %#v", root)
	}
	return root
}

func assertSameRoot(t *testing.T, a, b domain.ProviderRoot) {
	t.Helper()
	if a.PhysicalIdentity != b.PhysicalIdentity || a.FallbackIdentity != b.FallbackIdentity {
		t.Fatalf("different root identity:\n%#v\n%#v", a, b)
	}
}

func shortPath(path string) (string, error) {
	input, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	size := uint32(256)
	for {
		buffer := make([]uint16, size)
		n, err := windows.GetShortPathName(input, &buffer[0], size)
		if err != nil {
			return "", err
		}
		if n < size {
			return windows.UTF16ToString(buffer[:n]), nil
		}
		size = n + 1
	}
}
