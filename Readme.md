
# ichijiku イチジク 

Docker orchestration tool compatible with [fig](http://www.fig.sh) written in Go

Uses the docker cli client instead of the docker remote api in order to have access to all the latest features

## install

```bash
 $ go get github.com/tomerdmnt/ichijiku
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
```

[additional info](http://www.fig.sh/yml.html)

## Status

All commands are implemented (but not all flags)

fig.yml domainname, dns and volumes_from fields are not yet implemented
