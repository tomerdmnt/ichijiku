package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
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
}

// init fields not found in the yaml file
func (s *service) init(name string) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	s.name = name
	s.namespace = path.Base(dir)
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
	return cmd.Run()
}

func (s *service) runCmd(daemon, verbose bool) error {
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
	cmd := exec.Command("docker", args...)
	if verbose {
		fmt.Printf("%s\n", strings.Trim(fmt.Sprint(cmd.Args), "[]"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if !daemon {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

// returns true if container belongs to this service
func (s *service) matchContainer(container string) bool {
	return s.containerRe.MatchString(container)
}

func (s *service) String() string {
	return s.namespace + "/" + s.name
}
