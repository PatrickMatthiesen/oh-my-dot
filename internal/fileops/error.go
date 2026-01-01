package fileops

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
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
	ColorPrint(fmt.Sprintf("error: %s", err), Red)
	ColorPrint(message, Yellow)

	os.Exit(1)
}

func ColorPrint(message string, color string) {
	if shouldUseColor() {
		fmt.Printf("%s%s\x1b[0m", color, message)
	} else {
		fmt.Print(message)
	}
}

func ColorPrintln(message string, color string) {
	if shouldUseColor() {
		fmt.Printf("%s%s\x1b[0m\n", color, message)
	} else {
		fmt.Println(message)
	}
}

func ColorPrintf(color string, format string, a ...any) {
	if false {
		_ = fmt.Sprintf(format, a...) // enable printf analyser
	}
	if shouldUseColor() {
		fmt.Printf(color+format+"\x1b[0m", a...)
	} else {
		fmt.Printf(format, a...)
	}
}

func ColorPrintfn(color string, format string, a ...any) {
	if false {
		_ = fmt.Sprintf(format, a...) // enable printf analyser
	}
	if shouldUseColor() {
		fmt.Printf(color+format+"\x1b[0m\n", a...)
	} else {
		fmt.Printf(format+"\n", a...)
	}
}

func SColorPrint(message string, color string) string {
	if shouldUseColor() {
		return fmt.Sprintf("%s%s\x1b[0m", color, message)
	}
	return message
}

func SColorPrintln(message string, color string) string {
	if shouldUseColor() {
		return fmt.Sprintf("%s%s\x1b[0m\n", color, message)
	}
	return message + "\n"
}

func SColorPrintf(format string, a ...any) string {
	if false {
		_ = fmt.Sprintf(format, a...) // enable printf analyser
	}
	return fmt.Sprintf(format+"\x1b[0m", a...)
}

const (
	Red    string = "\x1b[31;1m"
	Green  string = "\x1b[32;1m"
	Yellow string = "\x1b[33;1m"
	Blue   string = "\x1b[34;1m"
	Purple string = "\x1b[35;1m"
	Cyan   string = "\x1b[36;1m"
	White  string = "\x1b[37;1m"

	WeirdColor string = "\x1b[38;2;255;102;153m"

	Reset string = "\x1b[0m"
)

func SColor(message string) string {
	// TODO: should provide a way to specify a format which allows specifying colors by name, e.g. "red", "green", etc.
	// the color should be easy to apply either the entire string or sections of the string
	// meaning multiple colors can be applied to the same string for different sections/words
	return message
}

// shouldUseColor determines if color output should be used
// based on NO_COLOR environment variable and TTY detection
func shouldUseColor() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if stdout is a TTY
	return isatty.IsTerminal(os.Stdout.Fd())
}