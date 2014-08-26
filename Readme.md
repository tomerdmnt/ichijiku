
# ichijiku イチジク 

Docker orchestration tool compatible with [fig](http://www.fig.sh) written in Go

Uses the docker command line client instead of the docker remote api in order to have access to all the latest features

## install

```bash
 $ go get github.com/tomerdmnt/ichijiku
```

[binaries](http://github.com/tomerdmnt/ichijiku/releases)

## Usage

```
 $ ichijiku -h

NAME:
   ichijiku - fig like docker orchestration

USAGE:
   ichijiku [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
   build	Build or rebuild services
   ps		List containers
   logs		View output from containers
   up		Build, (re)create, start and attach to containers for a service.
   rm		Remove all service containers.
   scale	Set number of containers to run for a service.
   start	Start existing containers.
   stop		Stop existing containers.
   kill		Force stop service containers.
   help, h	Shows a list of commands or help for one command
   
GLOBAL OPTIONS:
   --file, -f 'fig.yml'		Specify an alternate fig file
   --verbose, -V		Verbose output
   --help, -h			show help
   --generate-bash-completion	
   --version, -v		print the version
```

use -V/--verbose to see all docker commands that are run

```bash
 $ ichijiku -V run -d
```
   
## fig.yml

Uses [docker/fig](docker/fig) fig.yml

Example fig.yml:
```yml
web:
  build: .
  command: mon "node ./app.js"
  links:
    - db
  ports:
    - "9000:9000"
  environment:
    PORT: 9000
	KEY: ABCDEFGH
db:
  image: klaemo/couchdb
  ports:
    - "5984"
  volumes:
	- /local/path:/container/path
  run_flags: --restart=always
```

supports latest docker run features using run_flags yaml field

```yml
  run_flags: --restart=always
```

[additional info](http://www.fig.sh/yml.html)

## Status

All commands are implemented (but not all flags)

fig.yml domainname, dns and volumes_from fields are not yet implemented
