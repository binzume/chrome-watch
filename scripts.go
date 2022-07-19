package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type UserScript struct {
	Name       string
	Matches    []string
	ScriptPath string
}

func (s *UserScript) Match(url string) bool {
	for _, ptn := range s.Matches {
		if strings.HasSuffix(ptn, "*") {
			if strings.HasPrefix(url, strings.TrimSuffix(ptn, "*")) {
				return true
			}
		} else if url == ptn {
			return true
		}
	}
	return false
}

func (s *UserScript) Read() ([]byte, error) {
	return ioutil.ReadFile(s.ScriptPath)
}

func ScanUserScript(dir string) []*UserScript {
	files, _ := filepath.Glob(dir + "/*.user.js")
	var scripts []*UserScript
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			log.Println("Failed to read ", file)
			continue
		}
		defer f.Close()

		script := &UserScript{
			Name:       strings.TrimSuffix(filepath.Base(file), ".user.js"),
			ScriptPath: file,
		}
		reader := bufio.NewReaderSize(f, 1024)
		for line := ""; err == nil; line, err = reader.ReadString('\n') {
			if strings.Contains(line, "==/UserScript==") {
				break
			}
			if row := strings.Fields(strings.TrimSpace(line)); len(row) >= 3 && row[0] == "//" {
				if row[1] == "@name" {
					script.Name = row[2]
				} else if row[1] == "@match" {
					script.Matches = append(script.Matches, row[2])
				}
			}
		}
		scripts = append(scripts, script)
	}
	return scripts
}
