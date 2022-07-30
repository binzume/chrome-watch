package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type UserScript struct {
	Name       string
	Includes   []string
	Excludes   []string
	Grants     map[string]struct{}
	ScriptPath string
}

func (s *UserScript) Match(url string) bool {
	return s.PatternMatch(s.Includes, url) && !s.PatternMatch(s.Excludes, url)
}

func (s *UserScript) PatternMatch(ptns []string, str string) bool {
	for _, ptn := range ptns {
		if strings.HasSuffix(ptn, "*") {
			if strings.HasPrefix(str, strings.TrimSuffix(ptn, "*")) {
				return true
			}
		} else if str == ptn {
			return true
		}
	}
	return false
}

func (s *UserScript) Read() ([]byte, error) {
	return ioutil.ReadFile(s.ScriptPath)
}

var scriptAttrRe = regexp.MustCompile(`@([a-z]+)\s+(.*)`)

func parseScript(file string) (*UserScript, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	script := &UserScript{
		Name:       strings.TrimSuffix(filepath.Base(file), ".user.js"),
		ScriptPath: file,
		Grants:     map[string]struct{}{},
	}
	reader := bufio.NewReaderSize(f, 1024)
	for line := ""; err == nil; line, err = reader.ReadString('\n') {
		if strings.Contains(line, "==/UserScript==") {
			break
		}
		m := scriptAttrRe.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) > 2 {
			if m[1] == "name" {
				script.Name = m[2]
			} else if m[1] == "match" || m[1] == "include" {
				script.Includes = append(script.Includes, m[2])
			} else if m[1] == "exclude" {
				script.Excludes = append(script.Excludes, m[2])
			} else if m[1] == "grant" {
				script.Grants[m[2]] = struct{}{}
			}
		}
	}
	return script, nil
}

func ScanUserScript(dir string) []*UserScript {
	files, _ := filepath.Glob(dir + "/*.user.js")
	var scripts []*UserScript
	for _, file := range files {
		script, err := parseScript(file)
		if err != nil {
			log.Println("failed to load script ", file, err)
		}
		scripts = append(scripts, script)
	}
	return scripts
}
