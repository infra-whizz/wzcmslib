module github.com/infra-whizz/wzcmslib

go 1.13

require (
	github.com/antonfisher/nested-logrus-formatter v1.0.3
	github.com/bramvdbogaerde/go-scp v0.0.0-20200119201711-987556b8bdd7
	github.com/davecgh/go-spew v1.1.1
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/google/uuid v1.2.0
	github.com/infra-whizz/wzbox v0.0.0-20210223141646-d2405805b379 // indirect
	github.com/infra-whizz/wzlib v0.0.0-20200622182529-c99727f3707a
	github.com/karrick/godirwalk v1.15.6
	github.com/kr/text v0.2.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/sirupsen/logrus v1.5.0
	github.com/stretchr/objx v0.1.1 // indirect
	github.com/thoas/go-funk v0.7.0
	go.starlark.net v0.0.0-20200305232040-dcffbd0efcc1
	golang.org/x/crypto v0.0.0-20200219234226-1ad67e1f0ef4
	golang.org/x/sys v0.0.0-20200302150141-5c8b2ff67527
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

replace github.com/infra-whizz/wzlib => ../wzlib
