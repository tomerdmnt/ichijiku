package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kballard/go-shellquote"
)

type container struct {
	name    string
	ports   string
	status  string
	command string
	service *service
	index   int
}

func newContainer(s *service, i int) *container {
	name := fmt.Sprintf("%s_%s_%d", s.namespace, s.name, i)
	return &container{name: name, service: s, index: i}
}

func newContainerFromPsData(s *service, psd *psData) (*container, error) {
	fields := strings.Split(psd.name, "_")
	i, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil {
		return nil, err
	}
	return &container{
		name:    psd.name,
		ports:   psd.ports,
		status:  psd.status,
		command: psd.command,
		service: s,
		index:   i,
	}, nil
}

func (c *container) run(logsCh chan<- string, cp *colorPicker, daemon, verbose bool) error {
	cmd, err := c.buildRunCmd(daemon)
	if err != nil {
		return err
	}
	if !daemon {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		stdouterr := io.MultiReader(stdout, stderr)
		logsprefix := fmt.Sprintf("%s_%d", c.service.name, c.index)
		go processLogs(logsprefix, stdouterr, logsCh, cp)
	}
	c.rmf(verbose)
	if verbose {
		fmt.Printf("%s\n", strings.Trim(fmt.Sprint(cmd.Args), "[]"))
		if daemon {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func (c *container) logs(ch chan<- string, cp *colorPicker, timestamps, verbose bool) error {
	args := []string{"logs", "-f"}
	if timestamps {
		args = append(args, "-t")
	}
	args = append(args, c.name)

	cmd := exec.Command("docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if verbose {
		fmt.Printf("%s\n", strings.Trim(fmt.Sprint(cmd.Args), "[]"))
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	stdouterr := io.MultiReader(stdout, stderr)
	logsprefix := fmt.Sprintf("%s_%d", c.service.name, c.index)
	go processLogs(logsprefix, stdouterr, ch, cp)
	return nil
}

func processLogs(name string, r io.Reader, ch chan<- string, cp *colorPicker) {
	color := cp.next()
	reset := cp.reset()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := fmt.Sprintf(color+"%15s  | %s"+reset, name, scanner.Text())
		ch <- line
	}
}

func (c *container) buildRunCmd(daemon bool) (*exec.Cmd, error) {
	args := []string{"run"}
	if daemon {
		args = append(args, "-d")
	}
	args = append(args, fmt.Sprintf("--name=%s", c.name))
	for _, v := range c.service.Volumes {
		args = append(args, fmt.Sprintf("--volume=%s", v))
	}
	for _, p := range c.service.Ports {
		args = append(args, fmt.Sprintf("--publish=%s", p))
	}
	for env, val := range c.service.Environment {
		args = append(args, fmt.Sprintf("--env=\"%s=%s\"", env, val))
	}
	if c.service.Image == "" {
		args = append(args, c.service.String())
	} else {
		args = append(args, c.service.Image)
	}
	words, err := shellquote.Split(c.service.Command)
	if err != nil {
		return nil, err
	}
	args = append(args, words...)
	return exec.Command("docker", args...), nil
}

func (c *container) rmf(verbose bool) {
	cmd := exec.Command("docker", "rm", "-f", c.name)
	if verbose {
		fmt.Printf("%s\n", strings.Trim(fmt.Sprint(cmd.Args), "[]"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Run()
}

// Sort interface
type ByIndex []*container

func (a ByIndex) Len() int {
	return len(a)
}

func (a ByIndex) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByIndex) Less(i, j int) bool {
	return a[i].index < a[j].index
}
