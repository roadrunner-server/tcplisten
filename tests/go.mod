module github.com/roadrunner-server/tcplisten/tests

go 1.27

toolchain go1.27.1

require (
	github.com/roadrunner-server/tcplisten v1.5.2
	github.com/stretchr/testify v1.12.1
)

replace github.com/roadrunner-server/tcplisten => ../

require go.yaml.in/yaml/v3 v3.0.5 // indirect
