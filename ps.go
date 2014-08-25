package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type psData struct {
	name    string
	ports   string
	status  string
	command string
}

func ps(ch chan<- *psData, verbose bool) error {
	cmd := exec.Command("docker", "ps", "-a")
	if verbose {
		fmt.Printf("%s\n", strings.Trim(fmt.Sprint(cmd.Args), "[]"))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		fieldsRe := regexp.MustCompile("\\s{2,}")
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			d := &psData{}
			fields := fieldsRe.Split(scanner.Text(), -1)
			if len(fields) >= 7 {
				if fields[6] == "" {
					d.name = fields[5]
				} else {
					d.name = fields[6]
					d.ports = fields[5]
				}
			} else if len(fields) == 6 {
				d.name = fields[5]
				if fields[5] == "" {
					d.name = fields[4]
				}
			} else {
				continue
			}
			d.name = strings.Split(d.name, ",")[0]
			d.status = fields[4]
			d.command = fields[2]
			ch <- d
		}
		close(ch)
	}()
	return nil
}
