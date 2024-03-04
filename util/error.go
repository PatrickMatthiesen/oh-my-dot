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
	fmt.Printf("%s%s\x1b[0m\n", color, message)
}

func SColorPrint(message string, color string) string {
	return fmt.Sprintf(color, message, "\n")
}


const (
	Red    string = "\x1b[31;1m"
	Green  string = "\x1b[32;1m"
	Yellow string = "\x1b[33;1m"
	Blue   string = "\x1b[34;1m"
	Purple string = "\x1b[35;1m"
	Cyan   string = "\x1b[36;1m"
	White  string = "\x1b[37;1m"

	Reset string = "\x1b[0m"
)