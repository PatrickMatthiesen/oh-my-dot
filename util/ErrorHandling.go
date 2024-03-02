package util

import (
	"fmt"
	"os"
)

func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}

func CheckIfErrorWithMessage(err error, message string) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err), message)
	os.Exit(1)
}

func ColorPrint(message string, color string) {
	//TODO: use a color enum, and use that instead of the hardcoded string
	fmt.Printf(color, message)
}

func SColorPrint(message string, color string) string {
	return fmt.Sprintf(color, message)
}