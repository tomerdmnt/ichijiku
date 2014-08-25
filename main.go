package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/tabwriter"

	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v1"
)

func buildCmd(services []service, c *cli.Context) {
	for _, s := range services {
		if err := s.buildCmd(c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
	}
}

func psCmd(services []service, c *cli.Context) {
	containers := make(chan container)
	if err := ps(containers); err != nil {
		log.Fatal(err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "NAME\tCOMMAND\tSTATE\tPORTS")

	for c := range containers {
		for _, s := range services {
			if s.matchContainer(c.name) {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", c.name, c.command, c.status, c.ports)
				break
			}
		}
	}
	w.Flush()
}

func logsCmd(services []service, c *cli.Context) {
	ch := make(chan string)
	cp := newColorPicker()
	for _, s := range services {
		s.logs(ch, cp, c.GlobalBool("verbose"))
	}
	for line := range ch {
		fmt.Println(line)
	}
}

func upCmd(services []service, c *cli.Context) {
	daemon := c.Bool("d")
	verbose := c.GlobalBool("verbose")
	cp := newColorPicker()
	var logsCh chan string = nil
	if !daemon {
		logsCh = make(chan string)
	}
	for _, s := range services {
		if err := s.rm(verbose); err != nil {
			log.Fatal(err)
		}
		if err := s.runCmd(logsCh, cp, daemon, verbose); err != nil {
			log.Fatal(err)
		}
	}
	if logsCh != nil {
		for line := range logsCh {
			fmt.Println(line)
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

func createAction(action func([]service, *cli.Context)) func(*cli.Context) {
	return func(c *cli.Context) {
		serviceMap, err := parseFile(c.GlobalString("file"))
		if err != nil {
			log.Fatal(err)
		}
		services := []service{}
		if len(c.Args()) == 0 {
			for name, s := range serviceMap {
				s.init(name)
				services = append(services, s)
			}
		} else {
			for _, name := range c.Args() {
				s, ok := serviceMap[name]
				if !ok {
					log.Fatalf("%s: service does not exist", name)
				}
				s.init(name)
				services = append(services, s)
			}
		}
		action(services, c)
	}
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
			Action: createAction(buildCmd),
		},
		{
			Name:   "ps",
			Usage:  "List containers",
			Action: createAction(psCmd),
		},
		{
			Name:   "logs",
			Usage:  "View output from containers",
			Action: createAction(logsCmd),
		},
		{
			Name:   "up",
			Usage:  "Build, (re)create, start and attach to containers for a service.",
			Action: createAction(upCmd),
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "d",
					Usage: "Detached mode: Run containers in the background",
				},
			},
		},
	}
	app.Run(os.Args)
}
