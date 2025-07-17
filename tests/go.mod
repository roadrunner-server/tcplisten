module github.com/roadrunner-server/tcplisten/tests

go 1.24

toolchain go1.24.0

require (
	github.com/roadrunner-server/tcplisten v1.5.2
	github.com/stretchr/testify v1.10.0
)

replace github.com/roadrunner-server/tcplisten => ../

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
