package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

type config struct {
	FileName    string `env:"file_name,required"`
	FileContent string `env:"file_content,required"`
	UseSudo     string `env:"use_sudo,opt[yes,no]"`
}

func fail(format string, v ...any) {
	log.Errorf(format, v...)
	os.Exit(1)
}

// writeFile writes content to path. It always tries as the current user first;
// if that fails and sudo is enabled, it logs the failure and retries with sudo
// (required for privileged, root-owned locations on stacks where the build runs
// as a non-root user). This keeps the generated file user-owned whenever
// possible and only escalates to root when necessary.
func writeFile(path, content string, sudoEnabled bool) error {
	err := fileutil.WriteStringToFile(path, content)
	if err == nil {
		return nil
	}
	if !sudoEnabled {
		return err
	}

	log.Warnf("Failed to write file without sudo: %s", err)
	log.Printf("Retrying with sudo...")

	return writeFileWithSudo(path, content)
}

// writeFileWithSudo writes content to path as root, creating parent directories
// if needed. This is required to generate files in privileged (root-owned)
// locations on stacks where the build runs as a non-root user. sudo is run with
// -n (non-interactive) so it fails fast instead of waiting for a password when
// passwordless sudo is unavailable.
func writeFileWithSudo(path, content string) error {
	dir := filepath.Dir(path)
	mkdir := command.New("sudo", "-n", "mkdir", "-p", dir).
		SetStdout(os.Stdout).
		SetStderr(os.Stderr)
	if err := mkdir.Run(); err != nil {
		return fmt.Errorf("create parent directory (%s) with sudo: %w", dir, err)
	}

	tee := command.New("sudo", "-n", "tee", path).
		SetStdin(strings.NewReader(content)).
		SetStdout(io.Discard).
		SetStderr(os.Stderr)
	if err := tee.Run(); err != nil {
		return fmt.Errorf("write file with sudo: %w", err)
	}

	return nil
}

func main() {
	var cfg config
	if err := stepconf.Parse(&cfg); err != nil {
		fail("Issue with input: %s", err)
	}
	stepconf.Print(cfg)
	fmt.Println()

	absFilePath, err := pathutil.AbsPath(cfg.FileName)
	if err != nil {
		fail("Failed to determine absolute path of file (%s): %s", cfg.FileName, err)
	}

	if err := writeFile(absFilePath, cfg.FileContent, cfg.UseSudo == "yes"); err != nil {
		fail("Failed to write into file (%s): %s", absFilePath, err)
	}

	if err := tools.ExportEnvironmentWithEnvman("GENERATED_TEXT_FILE_PATH", absFilePath); err != nil {
		fail("Failed to export output (GENERATED_TEXT_FILE_PATH): %s", err)
	}

	log.Donef("The generated text file is available at: %s", absFilePath)
}
