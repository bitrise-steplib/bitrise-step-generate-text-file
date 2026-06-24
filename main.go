package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

type config struct {
	FileName    string `env:"file_name,required"`
	FileContent string `env:"file_content,required"`
}

func fail(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
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

	if err := fileutil.WriteStringToFile(absFilePath, cfg.FileContent); err != nil {
		fail("Failed to write into file (%s): %s", absFilePath, err)
	}

	if err := tools.ExportEnvironmentWithEnvman("GENERATED_TEXT_FILE_PATH", absFilePath); err != nil {
		fail("Failed to export output (GENERATED_TEXT_FILE_PATH): %s", err)
	}

	log.Donef("The generated text file is available at: %s", absFilePath)
}
