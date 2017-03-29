// +build windows

package fasthttp

import "testing"

func TestURIPathNormalizeIssue86(t *testing.T) {
	// see https://github.com/aresgo/router/fasthttp/issues/86
	var u URI

	testURIPathNormalize(t, &u, `C:\a\b\c\fs.go`, `C:\a\b\c\fs.go`)
}
