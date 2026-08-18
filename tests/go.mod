module github.com/roadrunner-server/tcplisten/tests

go 1.26

toolchain go1.26.6

require (
	github.com/roadrunner-server/tcplisten v1.5.2
	github.com/stretchr/testify v1.12.0
)

replace github.com/roadrunner-server/tcplisten => ../

require gopkg.in/yaml.v3 v3.0.1 // indirect
