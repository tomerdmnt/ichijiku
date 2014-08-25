package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
)

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
	containers  []*container
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
	s.containers = []*container{}
}

func (s *service) build(verbose bool) error {
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

func (s *service) run(logsCh chan<- string, cp *colorPicker, daemon, verbose bool) error {
	logfmt := "recreating %s...\n"
	if len(s.containers) == 0 {
		c := newContainer(s, 1)
		s.containers = append(s.containers, c)
		logfmt = "creating %s...\n"
	}
	for _, c := range s.containers {
		fmt.Printf(logfmt, c.name)
		err := c.run(logsCh, cp, daemon, verbose)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *service) logs(ch chan<- string, cp *colorPicker, timestamps, verbose bool) (int, error) {
	count := 0
	for _, c := range s.containers {
		err := c.logs(ch, cp, timestamps, verbose)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *service) scale(n int, verbose bool) error {
	sort.Sort(ByIndex(s.containers))
	addContainer := func(i int) error {
		c := newContainer(s, i)
		s.containers = append(s.containers, c)
		fmt.Printf("starting %s...\n", c.name)
		err := c.run(nil, nil, true, verbose)
		if err != nil {
			return err
		}
		return nil
	}

	// add containers
	if n > len(s.containers) {
		left := n - len(s.containers)
		offset := 0
		for i, c := range s.containers {
			if i+1+offset != c.index {
				offset++
				if err := addContainer(i + 1); err != nil {
					return err
				}
				left -= 1
				if left <= 0 {
					return nil
				}
			}
		}
		for i := len(s.containers); left > 0; i, left = i+1, left-1 {
			if err := addContainer(i + 1); err != nil {
				return err
			}
		}
		// remove containers
	} else if n < len(s.containers) {
		for _, c := range s.containers[n:] {
			fmt.Printf("stopping %s...\n", c.name)
			c.rmf(verbose)
		}
	}
	return nil
}

func (s *service) rmf(verbose bool) {
	for _, c := range s.containers {
		fmt.Printf("removing %s...\n", c.name)
		c.rmf(verbose)
	}
}

// returns true if container belongs to this service
func (s *service) matchContainer(container string) bool {
	return s.containerRe.MatchString(container)
}

func (s *service) String() string {
	return s.namespace + "/" + s.name
}
