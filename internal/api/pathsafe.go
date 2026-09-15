package api

import (
	"path/filepath"
	"strings"
)

// resolveRootedPath confines reqPath under root, rejecting lexical
// traversal ("../..") and — since a lexical check alone can be bypassed
// by a symlink placed under root that points outside it — also resolving
// symlinks on the nearest existing ancestor and re-checking containment.
func resolveRootedPath(root, reqPath string) (string, bool) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}

	full := filepath.Join(absRoot, filepath.Clean("/"+reqPath))
	if full != absRoot && !strings.HasPrefix(full, absRoot+string(filepath.Separator)) {
		return "", false
	}

	if !symlinkContained(absRoot, full) {
		return "", false
	}
	return full, true
}

// symlinkContained walks up from path to the nearest existing ancestor,
// resolves symlinks on it, and confirms that resolved location is still
// under root. If nothing along the path exists yet, there is nothing a
// symlink could have redirected, so it's trivially contained.
func symlinkContained(root, path string) bool {
	dir := path
	for {
		resolved, err := filepath.EvalSymlinks(dir)
		if err == nil {
			return resolved == root || strings.HasPrefix(resolved, root+string(filepath.Separator))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return true
		}
		dir = parent
	}
}
