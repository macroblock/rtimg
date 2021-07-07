module github.com/macroblock/rtimg

go 1.15

replace github.com/macroblock/imed => ../../macroblock/imed
replace golang.com/x/ => ../../golang.com/x/

require (
	github.com/atotto/clipboard v0.1.4
	github.com/macroblock/imed v0.0.0-20210706100626-72995190d3f1
	github.com/malashin/go-ansi v0.0.0-20170109082841-516580d6516a
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
)
