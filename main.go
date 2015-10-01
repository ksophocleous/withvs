package main

import (
	"os"
	"os/exec"
	"github.com/joho/godotenv"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"os/signal"
	"github.com/Sirupsen/logrus"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &customTextFormatter{}
}

func findFlag(flags []string, flag string) bool {
	lflag := strings.ToLower(flag)
	for _, v := range flags {
		if (lflag == strings.ToLower(v)) {
			return true
		}
	}

	return false
}

func vsTools(flags []string) (string, string, error) {
	use_vs110 := findFlag(flags, "--vs11")
	use_vs120 := findFlag(flags, "--vs12")
	use_vs140 := findFlag(flags, "--vs14")
	
	if (!use_vs110 && !use_vs120 && !use_vs140) {
		use_vs110 = true
		use_vs120 = true
		use_vs140 = true
	}

	var toolset string

	if (use_vs140) {
		toolset = os.Getenv("VS140COMNTOOLS")
		if len(toolset) <= 0 {
			log.Info("vs140: appropriate version not found")
		} else {
			return toolset, "vs140", nil
		}
	}

	if (use_vs120) {
		toolset = os.Getenv("VS120COMNTOOLS")
		if len(toolset) <= 0 {
			log.Info("vs120: appropriate version not found")
		} else {
			return toolset, "vs120", nil
		}
	}

	if (use_vs110) {
		toolset = os.Getenv("VS110COMNTOOLS")
		if len(toolset) <= 0 {
			log.Info("vs110: appropriate version not found")
		} else {
			return toolset, "vs110", nil
		}
	}

	return "", "ukn", fmt.Errorf("VS Tools not found (withvs replies on environment variables VS*COMNTOOLS to find the visual studio)")
}

func vsConfigType(flags []string) string {
	use_32bit := findFlag(flags, "--32")

	if use_32bit {
		return "x86"
	} else {
		return "amd64"
	}
}

func containsAll(stack string, allNeedles []string) bool {
	for _, v := range allNeedles {
		if strings.Contains(stack, v) == false {
			return false
		}
	}

	return true
}

func cleanPath(flags []string) {
	path := os.Getenv("PATH")
	paths := strings.Split(path, ";")
	newPaths := []string{}

	for _, v := range paths {
		if containsAll(v, []string{"mingw64", "bin"}) == false {
			newPaths = append(newPaths, v)
		}
	}

	path = strings.Join(newPaths, ";")
	os.Setenv("PATH", path)
}

func executeComspec(flags[] string, batchFilename string) error {
	log.WithFields(logrus.Fields{"flags":flags, "batchFilename":batchFilename}).Info("executeComspec will try to execute")

	comspec := os.Getenv("COMSPEC")
	if len(comspec) <= 0 {
		return fmt.Errorf("environment variable COMSPEC not found. This is used to launch the batch file that will fetch env variables and it is needed")
	}
	return execute(flags, append([]string{comspec, "/c"}, batchFilename))
}

func execute(flags[] string, command []string) error {
	silentOnlyErrors := findFlag(flags, "--only-errors")

	cmd := exec.Command(command[0], command[1:]...)
	log.WithFields(logrus.Fields{"command":command}).Info("executing command")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(){
		_, ok := <-c
		if ok {
			cmd.Process.Kill()
		}
	}()

	if silentOnlyErrors {
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(string(out))
			return fmt.Errorf("command '%s' failed: %s", strings.Join(command, " "), err.Error())
		}
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to start the command '%s': %s", strings.Join(command, " "), err.Error())
		}
	}

	close(c)

	return nil
}

func main() {

	// TODO: check if os is windows

	args := os.Args[1:]
	i := 0
	flags := []string{}
	progArgs := []string{}

	for ;i < len(args); i++ {
		if args[i] == "--" {
			break
		}
	}

	flags = args[0:i]
	if i < len(args) {
		progArgs = args[i+1:]
	}

	vcToolsPath, toolsId, err := vsTools(flags)
	if err != nil {
		log.WithFields(logrus.Fields{"error":err}).Fatal("vsTools failed")
	}
	vcToolsPath = filepath.Join(vcToolsPath, "../../VC/vcvarsall.bat")

	vsConfigType := vsConfigType(flags)
	cleanPath(flags);

	home := os.Getenv("HOME")
	envFile := fmt.Sprintf("%s/withvs-%s-%s.env", home, toolsId, vsConfigType)

	saveenv := findFlag(flags, "--save-env")
	if saveenv {
		log.WithFields(logrus.Fields{"envFile":envFile}).Debug("Saving env to file")
		f, err := os.Create(envFile)
		if err != nil {
			log.WithFields(logrus.Fields{"error":err,"envFile":envFile}).Fatal("Error while trying to create environment cache file")
		}
		defer f.Close()
		for _, v := range os.Environ() {
			f.WriteString(fmt.Sprintf("%s\n", v))
		}
		return
	}

	err = godotenv.Overload(envFile)
	if err != nil {
		log.WithFields(logrus.Fields{"envFile":envFile}).Debug("could not find env file.. will try to create it")
		filename, err := ioutil.TempDir("", "")
		filename = fmt.Sprintf("%s\\_withvs_test.bat", filename)
		
		log.WithFields(logrus.Fields{"filename":filename}).Debug("creating batch file")
		f, err := os.Create(filename)
		if err != nil {
			log.WithFields(logrus.Fields{"error":err,"envFile":filename}).Fatal("failed to create temporary batch file")
		}

		f.WriteString(fmt.Sprintf("@call \"%s\" %s\n", vcToolsPath, vsConfigType))
		f.WriteString(fmt.Sprintf("@%s %s --save-env\n", os.Args[0], strings.Join(flags, " ")))
		f.Close()

		err = executeComspec(flags, filename)
		if err != nil {
			log.WithFields(logrus.Fields{"flags":flags,"error":err,"batchFilename":filename}).Fatal("failed to execute batchfile via comspec")
		}

		log.WithFields(logrus.Fields{"filename":filename, "toolset":toolsId, "config": vsConfigType}).Debug("saved env variables to disk")
	}

	err = godotenv.Overload(envFile)
	if err != nil {
		log.WithFields(logrus.Fields{"envFile": envFile, "error": err}).Fatal("Failed to load environment file even after it was created")
	}
	cleanPath(flags);

	verbose := findFlag(flags, "--verbose")
	if verbose {
		log.WithFields(logrus.Fields{"flags":flags}).Debug("Flags")
		log.WithFields(logrus.Fields{"progArgs":progArgs}).Debug("will launch this")
		log.WithFields(logrus.Fields{"vcToolsPath":vcToolsPath}).Debug("full path to VS tools")
		log.WithFields(logrus.Fields{"config":vsConfigType}).Debug("platform config")
		log.WithFields(logrus.Fields{"PATH": os.Getenv("PATH")}).Debug("full contents of PATH")
	}

	err = execute(flags, progArgs)
	if err != nil {
		log.WithFields(logrus.Fields{"flags":flags,"progArgs": progArgs,"error":err}).Fatal("Failed to execute command")
	}
}
