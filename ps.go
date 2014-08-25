package main

import (
	"bufio"
	"os/exec"
	"regexp"
	"strings"
)

type container struct {
	name    string
	ports   string
	status  string
	command string
}

func ps(containers chan<- container) error {
	cmd := exec.Command("docker", "ps", "-a")
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
			c := container{}
			fields := fieldsRe.Split(scanner.Text(), -1)
			if len(fields) >= 7 {
				if fields[6] == "" {
					c.name = fields[5]
				} else {
					c.name = fields[6]
					c.ports = fields[5]
				}
			} else if len(fields) == 6 {
				c.name = fields[5]
			} else {
				continue
			}
			c.name = strings.Split(c.name, ",")[0]
			c.status = fields[4]
			c.command = fields[2]
			containers <- c
		}
		close(containers)
	}()
	return nil
}
