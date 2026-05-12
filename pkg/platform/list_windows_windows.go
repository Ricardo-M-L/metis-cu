//go:build windows

package platform

// ListWindows on Windows — pending a Win32 EnumWindows + GetWindowText
// + GetWindowRect implementation via syscall. The path is well-
// understood (user32.dll EnumWindows callback) but it's ~80 lines of
// cgo / syscall plumbing, deferred until the macOS / Linux paths land
// in production and prove the tool shape.
//
// For now returns ErrNotImplemented so the list_windows tool surfaces
// a clean "list_windows backend not available on windows yet"
// message instead of a stub-zero list.
func (p *windowsPlatform) ListWindows() ([]WindowInfo, error) {
	return nil, ErrNotImplemented
}
