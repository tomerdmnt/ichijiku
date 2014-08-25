package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/kballard/go-shellquote"
)

var colors []string = []string{"cyan", "yellow", "green", "magenta", "blue", "red"}
var colori int = 0
var colorMutex sync.Mutex = sync.Mutex{}

type service struct {
	name        string
	namespace   string
	Build       string            `yaml:"build"`
	Command     string            `yaml:"command"`
	Image       string            `yaml:"image"`
	Ports       []string          `yaml:"ports"`
	Links       []string          `yaml:"links"`
	Environment map[string]string `yaml:"environment"`
	Volumes     []string          `yaml:"volumes"`
	containerRe *regexp.Regexp
}

// init fields not found in the yaml file
func (s *service) init(name string) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	s.name = name
	s.namespace = strings.Replace(path.Base(dir), "-", "", -1)
	s.containerRe = regexp.MustCompile(fmt.Sprintf("%s_%s_\\d+", s.namespace, s.name))
}

func (s *service) buildCmd(verbose bool) error {
	if s.Image != "" {
		fmt.Printf("%s uses image, skipping...\n", s.name)
		return nil
	}
	fmt.Printf("building %s\n", s)
	cmd := exec.Command("docker", "build", fmt.Sprintf("--tag=\"%s\"", s), s.Build)
	if verbose {
		fmt.Printf("%s\n", strings.Trim(fmt.Sprint(cmd.Args), "[]"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (s *service) rm(verbose bool) error {
	name := fmt.Sprintf("%s_%s_%d", s.namespace, s.name, 1)
	cmd := exec.Command("docker", "rm", "-f", name)
	if verbose {
		fmt.Printf("%s\n", strings.Trim(fmt.Sprint(cmd.Args), "[]"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Run()
	return nil
}

func (s *service) runCmd(logsCh chan<- string, cp *colorPicker, daemon, verbose bool) error {
	name := fmt.Sprintf("%s_%s_%d", s.namespace, s.name, 1)
	args := []string{"run"}
	if daemon {
		args = append(args, "-d")
	}
	args = append(args, fmt.Sprintf("--name=%s", name))
	for _, v := range s.Volumes {
		args = append(args, fmt.Sprintf("--volume=%s", v))
	}
	for _, p := range s.Ports {
		args = append(args, fmt.Sprintf("--publish=%s", p))
	}
	for env, val := range s.Environment {
		args = append(args, fmt.Sprintf("--env=\"%s=%s\"", env, val))
	}
	if s.Image == "" {
		args = append(args, s.String())
	} else {
		args = append(args, s.Image)
	}
	words, err := shellquote.Split(s.Command)
	if err != nil {
		return err
	}
	args = append(args, words...)

	cmd := exec.Command("docker", args...)
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
		go processLogs(name, stdouterr, logsCh, cp)
	}
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

func (s *service) logs(ch chan<- string, cp *colorPicker, verbose bool) error {
	containers := make(chan container)
	ps(containers)

	for c := range containers {
		if s.matchContainer(c.name) {
			cmd := exec.Command("docker", "logs", "-t", "-f", c.name)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				// TODO: cancel goroutine
				return err
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				// TODO: cancel goroutine
				return err
			}
			if verbose {
				fmt.Printf("%s\n", strings.Trim(fmt.Sprint(cmd.Args), "[]"))
			}
			if err := cmd.Start(); err != nil {
				return err
			}

			stdouterr := io.MultiReader(stdout, stderr)
			go processLogs(c.name, stdouterr, ch, cp)
		}
	}
	return nil
}

func processLogs(name string, r io.Reader, ch chan<- string, cp *colorPicker) {
	color := cp.next()
	reset := cp.reset()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := fmt.Sprintf(color+"%20s  | %s"+reset, name, scanner.Text())
		ch <- line
	}
}

// returns true if container belongs to this service
func (s *service) matchContainer(container string) bool {
	return s.containerRe.MatchString(container)
}

func (s *service) String() string {
	return s.namespace + "/" + s.name
}
