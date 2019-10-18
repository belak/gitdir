module github.com/belak/go-code

go 1.13

require (
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/gliderlabs/ssh v0.2.2
	github.com/libgit2/git2go v0.0.0-00010101000000-000000000000
	github.com/rs/zerolog v1.15.0
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.22.1
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	gopkg.in/yaml.v3 v3.0.0-20191010095647-fc94e3f71652
)

replace github.com/libgit2/git2go => github.com/belak/git2go v0.0.0-20191014155453-39f105e36806
