package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	bblfsh "gopkg.in/bblfsh/client-go.v3"
	"gopkg.in/bblfsh/client-go.v3/tools"
	"gopkg.in/bblfsh/sdk.v2/uast/nodes"
)

func main() {
	flag.Parse()

	client, err := bblfsh.NewClient("0.0.0.0:9432")
	if err != nil {
		panic(err)
	}
	defer client.Close()
	t0 := time.Now()

	for _, arg := range flag.Args() {
		fmt.Printf("// %s\n", filepath.Base(arg))
		req := client.NewParseRequest().Language("go").ReadFile(arg)
		uast, _, err := req.UAST()
		if err != nil {
			log.Println(err)
			continue
		}
		it, err := tools.Filter(uast, "//StructType")
		if err != nil {
			log.Println(err)
			continue
		}
		for it.Next() {
			extract(it.Node())
		}
	}
	fmt.Println(time.Now().Sub(t0))
}

func extract(node nodes.External) {
	switch n := (node).(type) {
	case nodes.Object:
		// fmt.Printf("%skeys: %v\n", prefix, n.Keys())

		if fields, ok := n["Fields"]; ok {
			extract(fields)
		}
		if list, ok := n["List"]; ok {
			extract(list)
		}
		if names, ok := n["Names"]; ok {
			extract(names)
			if typ, ok := n["Type"]; ok {
				extract(typ)
			}
			fmt.Println()
		}
		if name, ok := n["Name"]; ok {
			extract(name)
		}

	case nodes.Array:
		for _, nn := range n {
			extract(nn)
		}
	case nodes.String:
		fmt.Printf("\t%s", n.Native())
	case nodes.Int:
		fmt.Printf("\t%d", n.Native())
	case nodes.Uint:
		fmt.Printf("\t%d", n.Native())
	case nodes.Float:
		fmt.Printf("\t%f", n.Native())
	case nodes.Bool:
		fmt.Printf("\t%v", n.Native())
	}
}
