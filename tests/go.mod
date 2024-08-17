module github.com/roadrunner-server/tcplisten/tests

go 1.23.0

require (
	github.com/roadrunner-server/tcplisten v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.9.0
)

replace github.com/roadrunner-server/tcplisten => ../

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
