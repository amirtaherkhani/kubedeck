package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
	"golang.org/x/term"
)

func run(path string, ask func(string) (string, error), output io.Writer) error {
	if _, err := os.Lstat(path); err == nil {
		return errors.New("Infisical host env file already exists; setup will not overwrite it")
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot inspect Infisical host env file")
	}
	id, err := ask("Infisical Client ID: ")
	if err != nil {
		return errors.New("could not read Client ID from terminal")
	}
	secret, err := ask("Infisical Client Secret: ")
	if err != nil {
		return errors.New("could not read Client Secret from terminal")
	}
	if err := infisical.SaveHostCredentials(path, id, secret); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(output, "Private Infisical host configuration saved at %s. Restart the MCP stdio process to load it.\n", path)
	return nil
}

func terminalPrompt(label string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("terminal required")
	}
	_, _ = fmt.Fprint(os.Stderr, label)
	data, err := term.ReadPassword(fd)
	_, _ = fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	value := string(data)
	clear(data)
	return value, nil
}

func main() {
	path, err := infisical.HostEnvPath()
	if err == nil {
		err = run(path, terminalPrompt, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
