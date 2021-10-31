package cliutil

// Wrap the string `s` to a maximum width `w`.  Pass `w` == 0 to do no wrapping.
//
// In order to have some room for slop to avoid things like a short word being on a line by itself,
// most lines are actually wrapped to `w - 5`.
func Wrap(w int, s string) string {
	return wrap(0, w, s)
}

// Wrap the string `s` to a maximum width `w` with leading indent `i`.  The first line is not
// indented (this is assumed to be done by caller).  Pass `w` == 0 to do no wrapping
//
// In order to have some room for slop to avoid things like a short word being on a line by itself,
// most lines are actually wrapped to `w - 5`.
func WrapIndent(i, w int, s string) string {
	return wrap(i, w, s)
}
