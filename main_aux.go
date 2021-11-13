//go:build aux

package main

func init() {
	argparser.CompletionOptions.DisableDefaultCmd = false
}
