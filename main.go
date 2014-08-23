package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v1"
)

var (
	services []service
)

func before(c *cli.Context) error {
	serviceMap, err := parseFile(c.GlobalString("file"))
	if err != nil {
		log.Println(err)
		return err
	}
	services = []service{}
	if len(c.Args()) == 0 {
		for name, s := range serviceMap {
			s.init(name)
			services = append(services, s)
		}
	} else {
		for _, name := range c.Args() {
			s, ok := serviceMap[name]
			if !ok {
				err := fmt.Errorf("%s: service does not exist", name)
				log.Println(err)
				return err
			}
			s.init(name)
			services = append(services, s)
		}
	}
	return nil
}

func buildCmd(c *cli.Context) {
	for _, s := range services {
		if err := s.buildCmd(c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
	}
}

func psCmd(c *cli.Context) {
	cmd := exec.Command("docker", "ps", "-a")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(stdout)
	r := regexp.MustCompile("\\s{2,}")
	for scanner.Scan() {
		l := r.Split(scanner.Text(), -1)
		name := strings.Split(l[6], ",")[0]
		command := l[2]
		state := l[4]
		ports := l[5]
		if name == "" {
			name = ports
			ports = ""
		}
		for _, s := range services {
			if s.matchContainer(name) {
				fmt.Printf("%s\t\t%s\t\t\t\t%s\t\t%s\n", name, command, state, ports)
				break
			}
		}
	}
}

func logsCmd(c *cli.Context) {
}

func upCmd(c *cli.Context) {
	for _, s := range services {
		daemon := c.GlobalBool("d")
		verbose := c.GlobalBool("verbose")
		if err := s.rm(verbose); err != nil {
			log.Fatal(err)
		}
		if err := s.runCmd(daemon, verbose); err != nil {
			log.Fatal(err)
		}
	}
}

func parseFile(file string) (map[string]service, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	m := make(map[string]service)
	if err := yaml.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func main() {
	app := cli.NewApp()
	app.Name = "ichijiku"
	app.Usage = "fig like docker orchestration"
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "file, f",
			Value: "fig.yml",
			Usage: "Specify an alternate fig file",
		},
		cli.BoolFlag{
			Name:  "verbose, V",
			Usage: "Verbose output",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "build",
			Usage:  "Build or rebuild services",
			Before: before,
			Action: buildCmd,
		},
		{
			Name:   "ps",
			Usage:  "List containers",
			Before: before,
			Action: psCmd,
		},
		{
			Name:   "logs",
			Usage:  "View output from containers",
			Before: before,
			Action: logsCmd,
		},
		{
			Name:   "up",
			Usage:  "Build, (re)create, start and attach to containers for a service.",
			Before: before,
			Action: upCmd,
		},
	}
	app.Run(os.Args)
}
