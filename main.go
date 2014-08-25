package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v1"
)

func buildCmd(services []*service, c *cli.Context) {
	for _, s := range services {
		if err := s.build(c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
	}
}

func psCmd(services []*service, c *cli.Context) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "NAME\tCOMMAND\tSTATE\tPORTS")
	for _, s := range services {
		for _, cntr := range s.containers {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n",
				cntr.name, cntr.command, cntr.status, cntr.ports)
		}
	}
	w.Flush()
}

func logsCmd(services []*service, c *cli.Context) {
	ch := make(chan string)
	cp := newColorPicker()
	total := 0
	for _, s := range services {
		count, err := s.logs(ch, cp, c.Bool("timestamps"), c.GlobalBool("verbose"))
		if err != nil {
			log.Fatal(err)
		}
		total += count
	}
	if total > 0 {
		for line := range ch {
			fmt.Println(line)
		}
	}
}

func upCmd(services []*service, c *cli.Context) {
	daemon := c.Bool("d")
	verbose := c.GlobalBool("verbose")
	cp := newColorPicker()
	var logsCh chan string = nil
	if !daemon {
		logsCh = make(chan string)
	}

	sort.Sort(ByServiceDependency(services))
	for _, s := range services {
		if err := s.run(logsCh, cp, daemon, verbose); err != nil {
			log.Fatal(err)
		}
	}
	if logsCh != nil {
		for line := range logsCh {
			fmt.Println(line)
		}
	}
}

func rmCmd(services []*service, c *cli.Context) {
	for _, s := range services {
		s.rmf(c.GlobalBool("verbose"))
	}
}

func startCmd(services []*service, c *cli.Context) {
	for _, s := range services {
		if err := s.start(c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
	}
}

func stopCmd(services []*service, c *cli.Context) {
	for _, s := range services {
		if err := s.stop(c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
	}
}

func killCmd(services []*service, c *cli.Context) {
	for _, s := range services {
		if err := s.kill(c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
	}
}

func scaleCmd(c *cli.Context) {
	serviceMap, err := parseFile(c.GlobalString("file"))
	if err != nil {
		log.Fatal(err)
	}
	for name, s := range serviceMap {
		s.init(name, serviceMap)
	}

	// populate service containers
	psCh := make(chan *psData)
	if err := ps(psCh, c.GlobalBool("verbose")); err != nil {
		log.Fatal(err)
	}
	for psdata := range psCh {
		for _, s := range serviceMap {
			if s.matchContainer(psdata.name) {
				cntr, err := newContainerFromPsData(s, psdata)
				if err != nil {
					log.Fatal(err)
				}
				s.containers = append(s.containers, cntr)
				break
			}
		}
	}

	for _, arg := range c.Args() {
		fields := strings.Split(arg, "=")
		name := fields[0]
		n, err := strconv.Atoi(fields[1])
		if err != nil {
			log.Fatal(err)
		}
		s, ok := serviceMap[name]
		if !ok {
			log.Fatalf("%s: service does not exist", name)
		}

		err = s.scale(n, c.GlobalBool("verbose"))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func parseFile(file string) (map[string]*service, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*service)
	if err := yaml.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func createAction(action func([]*service, *cli.Context)) func(*cli.Context) {
	return func(c *cli.Context) {
		serviceMap, err := parseFile(c.GlobalString("file"))
		if err != nil {
			log.Fatal(err)
		}
		services := []*service{}
		if len(c.Args()) == 0 {
			for name, s := range serviceMap {
				s.init(name, serviceMap)
				services = append(services, s)
			}
		} else {
			for _, name := range c.Args() {
				s, ok := serviceMap[name]
				if !ok {
					log.Fatalf("%s: service does not exist", name)
				}
				s.init(name, serviceMap)
				services = append(services, s)
			}
		}
		psCh := make(chan *psData)
		if err := ps(psCh, c.GlobalBool("verbose")); err != nil {
			log.Fatal(err)
		}
		for psdata := range psCh {
			for _, s := range services {
				if s.matchContainer(psdata.name) {
					cntr, err := newContainerFromPsData(s, psdata)
					if err != nil {
						log.Fatal(err)
					}
					s.containers = append(s.containers, cntr)
					break
				}
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
	app.Version = "0.0.1"
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
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "timestamps, t",
					Usage: "Show timestamps",
				},
			},
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
		{
			Name:   "rm",
			Usage:  "Remove all service containers.",
			Action: createAction(rmCmd),
		},
		{
			Name:   "scale",
			Usage:  "Set number of containers to run for a service.",
			Description: "$ ichijiku scale web=3 db=2",
			Action: scaleCmd,
		},
		{
			Name:   "start",
			Usage:  "Start existing containers.",
			Action: createAction(startCmd),
		},
		{
			Name:   "stop",
			Usage:  "Stop existing containers.",
			Action: createAction(stopCmd),
		},
		{
			Name:   "kill",
			Usage:  "Force stop service containers.",
			Action: createAction(killCmd),
		},
	}
	app.Run(os.Args)
}
