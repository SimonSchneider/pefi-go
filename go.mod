module github.com/SimonSchneider/pefigo

go 1.23.0

require (
	github.com/SimonSchneider/goslu v0.1.4
	github.com/ncruces/go-sqlite3 v0.20.3
)

replace (
	github.com/SimonSchneider/goslu => ../goslu
)

require (
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.8.2 // indirect
	golang.org/x/sys v0.27.0 // indirect
)
