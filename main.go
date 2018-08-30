package main

import (
	"flag"
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		panic("some bullshit")
	}

	pkg := flag.Arg(0)
	conf := loader.Config{}
	conf.Import(pkg)
	prog, err := conf.Load()
	if err != nil {
		panic(err)
	}
	ssaprog := ssautil.CreateProgram(prog, ssa.GlobalDebug)
	ssaprog.Build()
	funcs := ssautil.AllFunctions(ssaprog)
	var ndefers, ndyndefers int
	for f := range funcs {
		// find all blocks containing defer instructions
		defers := make(map[*ssa.BasicBlock]*ssa.Defer)
		for _, b := range f.Blocks {
			for _, inst := range b.Instrs {
				def, ok := inst.(*ssa.Defer)
				if ok {
					defers[b] = def
					ndefers++
				}
			}
		}
		// figure out if the defer block can loop
		// TODO(dmo): probably some clever way of figuring this out
		// with the dominator tree info
	deferblock:
		for defblock, definst := range defers {
			seen := make(map[*ssa.BasicBlock]bool)
			var queue []*ssa.BasicBlock
			queue = append(queue, defblock)
			for len(queue) > 0 {
				curr := queue[len(queue)-1]
				queue = queue[:len(queue)-1]
				seen[curr] = true
				for _, succ := range curr.Succs {
					if succ == defblock {
						fmt.Println(prog.Fset.Position(definst.Pos()))
						ndyndefers++
						continue deferblock
					}
					if !seen[succ] {
						queue = append(queue, succ)
					}
				}
			}
		}
	}
	fmt.Println("program had", ndefers, "defers,", ndyndefers, "of which were dynamic")
}

type deferwalk struct {
	ndefers int
	path    []ast.Node
}

func (d *deferwalk) Visit(n ast.Node) ast.Visitor {
	_, ok := n.(*ast.DeferStmt)
	if !ok {
		return d
	}
	d.ndefers += 1
	return d
}
