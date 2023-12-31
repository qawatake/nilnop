// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Modifications Copyright (c) qawatake 2023

package nilnop

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"github.com/qawatake/nilnop/internal/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

const name = "nilnop"
const doc = "nilnop detects nil is passed to a function that does nothing for nil"
const url = "https://pkg.go.dev/github.com/qawatake/nilnop"

func NewAnalyzer(tgt ...Target) *analysis.Analyzer {
	a := &analyzer{
		targets: tgt,
	}
	return &analysis.Analyzer{
		Name:     name,
		Doc:      doc,
		URL:      url,
		Run:      a.run,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	}
}

type analyzer struct {
	targets []Target
}

func (a *analyzer) run(pass *analysis.Pass) (interface{}, error) {
	ssainput := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	v, err := newValidator(pass, a.targets)
	if err != nil {
		panic(err)
	}

	for _, fn := range ssainput.SrcFuncs {
		v.runFunc(pass, fn)
	}
	return nil, nil
}

func (a *validator) runFunc(pass *analysis.Pass, fn *ssa.Function) {
	reportf := func(category string, pos token.Pos, format string, args ...interface{}) {
		// We ignore nil-checking ssa.Instructions
		// that don't correspond to syntax.
		if pos.IsValid() {
			pass.Report(analysis.Diagnostic{
				Pos:      pos,
				Category: category,
				Message:  fmt.Sprintf(format, args...),
			})
		}
	}

	// visit visits reachable blocks of the CFG in dominance order,
	// maintaining a stack of dominating nilness facts.
	//
	// By traversing the dom tree, we can pop facts off the stack as
	// soon as we've visited a subtree.  Had we traversed the CFG,
	// we would need to retain the set of facts for each block.
	seen := make([]bool, len(fn.Blocks)) // seen[i] means visit should ignore block i
	var visit func(b *ssa.BasicBlock, stack []fact)
	visit = func(b *ssa.BasicBlock, stack []fact) {
		if seen[b.Index] {
			return
		}
		seen[b.Index] = true
		// Report nil dereferences.
		for _, instr := range b.Instrs {
			if instr, ok := instr.(ssa.CallInstruction); ok {
				a.validate(stack, instr)
			}
		}

		// For nil comparison blocks, report an error if the condition
		// is degenerate, and push a nilness fact on the stack when
		// visiting its true and false successor blocks.
		if binop, tsucc, fsucc := eq(b); binop != nil {
			xnil := nilnessOf(stack, binop.X)
			ynil := nilnessOf(stack, binop.Y)

			if ynil != unknown && xnil != unknown && (xnil == isnil || ynil == isnil) {
				// Degenerate condition:
				// the nilness of both operands is known,
				// and at least one of them is nil.
				var adj string
				if (xnil == ynil) == (binop.Op == token.EQL) {
					adj = "tautological"
				} else {
					adj = "impossible"
				}
				reportf("cond", binop.Pos(), "%s condition: %s %s %s", adj, xnil, binop.Op, ynil)

				// If tsucc's or fsucc's sole incoming edge is impossible,
				// it is unreachable.  Prune traversal of it and
				// all the blocks it dominates.
				// (We could be more precise with full dataflow
				// analysis of control-flow joins.)
				var skip *ssa.BasicBlock
				if xnil == ynil {
					skip = fsucc
				} else {
					skip = tsucc
				}
				for _, d := range b.Dominees() {
					if d == skip && len(d.Preds) == 1 {
						continue
					}
					visit(d, stack)
				}
				return
			}

			// "if x == nil" or "if nil == y" condition; x, y are unknown.
			if xnil == isnil || ynil == isnil {
				var newFacts facts
				if xnil == isnil {
					// x is nil, y is unknown:
					// t successor learns y is nil.
					newFacts = expandFacts(fact{binop.Y, isnil})
				} else {
					// x is nil, y is unknown:
					// t successor learns x is nil.
					newFacts = expandFacts(fact{binop.X, isnil})
				}

				for _, d := range b.Dominees() {
					// Successor blocks learn a fact
					// only at non-critical edges.
					// (We could do be more precise with full dataflow
					// analysis of control-flow joins.)
					s := stack
					if len(d.Preds) == 1 {
						if d == tsucc {
							s = append(s, newFacts...)
						} else if d == fsucc {
							s = append(s, newFacts.negate()...)
						}
					}
					visit(d, s)
				}
				return
			}
		}

		for _, d := range b.Dominees() {
			visit(d, stack)
		}
	}

	// Visit the entry block.  No need to visit fn.Recover.
	if fn.Blocks != nil {
		visit(fn.Blocks[0], make([]fact, 0, 20)) // 20 is plenty
	}
}

// A fact records that a block is dominated
// by the condition v == nil or v != nil.
type fact struct {
	value   ssa.Value
	nilness nilness
}

func (f fact) negate() fact { return fact{f.value, -f.nilness} }

type nilness int

const (
	isnonnil         = -1
	unknown  nilness = 0
	isnil            = 1
)

var nilnessStrings = []string{"non-nil", "unknown", "nil"}

func (n nilness) String() string { return nilnessStrings[n+1] }

// nilnessOf reports whether v is definitely nil, definitely not nil,
// or unknown given the dominating stack of facts.
func nilnessOf(stack []fact, v ssa.Value) nilness {
	switch v := v.(type) {
	// unwrap ChangeInterface and Slice values recursively, to detect if underlying
	// values have any facts recorded or are otherwise known with regard to nilness.
	//
	// This work must be in addition to expanding facts about
	// ChangeInterfaces during inference/fact gathering because this covers
	// cases where the nilness of a value is intrinsic, rather than based
	// on inferred facts, such as a zero value interface variable. That
	// said, this work alone would only inform us when facts are about
	// underlying values, rather than outer values, when the analysis is
	// transitive in both directions.
	case *ssa.ChangeInterface:
		if underlying := nilnessOf(stack, v.X); underlying != unknown {
			return underlying
		}
	case *ssa.Slice:
		if underlying := nilnessOf(stack, v.X); underlying != unknown {
			return underlying
		}
	case *ssa.SliceToArrayPointer:
		nn := nilnessOf(stack, v.X)
		if slice2ArrayPtrLen(v) > 0 {
			if nn == isnil {
				// We know that *(*[1]byte)(nil) is going to panic because of the
				// conversion. So return unknown to the caller, prevent useless
				// nil deference reporting due to * operator.
				return unknown
			}
			// Otherwise, the conversion will yield a non-nil pointer to array.
			// Note that the instruction can still panic if array length greater
			// than slice length. If the value is used by another instruction,
			// that instruction can assume the panic did not happen when that
			// instruction is reached.
			return isnonnil
		}
		// In case array length is zero, the conversion result depends on nilness of the slice.
		if nn != unknown {
			return nn
		}
	}

	// Is value intrinsically nil or non-nil?
	switch v := v.(type) {
	case *ssa.Alloc,
		*ssa.FieldAddr,
		*ssa.FreeVar,
		*ssa.Function,
		*ssa.Global,
		*ssa.IndexAddr,
		*ssa.MakeChan,
		*ssa.MakeClosure,
		*ssa.MakeInterface,
		*ssa.MakeMap,
		*ssa.MakeSlice:
		return isnonnil
	case *ssa.Const:
		if v.IsNil() {
			return isnil // nil or zero value of a pointer-like type
		} else {
			return unknown // non-pointer
		}
	}

	// Search dominating control-flow facts.
	for _, f := range stack {
		if f.value == v {
			return f.nilness
		}
	}
	return unknown
}

func slice2ArrayPtrLen(v *ssa.SliceToArrayPointer) int64 {
	return v.Type().(*types.Pointer).Elem().Underlying().(*types.Array).Len()
}

// If b ends with an equality comparison, eq returns the operation and
// its true (equal) and false (not equal) successors.
func eq(b *ssa.BasicBlock) (op *ssa.BinOp, tsucc, fsucc *ssa.BasicBlock) {
	if If, ok := b.Instrs[len(b.Instrs)-1].(*ssa.If); ok {
		if binop, ok := If.Cond.(*ssa.BinOp); ok {
			switch binop.Op {
			case token.EQL:
				return binop, b.Succs[0], b.Succs[1]
			case token.NEQ:
				return binop, b.Succs[1], b.Succs[0]
			}
		}
	}
	return nil, nil, nil
}

// expandFacts takes a single fact and returns the set of facts that can be
// known about it or any of its related values. Some operations, like
// ChangeInterface, have transitive nilness, such that if you know the
// underlying value is nil, you also know the value itself is nil, and vice
// versa. This operation allows callers to match on any of the related values
// in analyses, rather than just the one form of the value that happened to
// appear in a comparison.
//
// This work must be in addition to unwrapping values within nilnessOf because
// while this work helps give facts about transitively known values based on
// inferred facts, the recursive check within nilnessOf covers cases where
// nilness facts are intrinsic to the underlying value, such as a zero value
// interface variables.
//
// ChangeInterface is the only expansion currently supported, but others, like
// Slice, could be added. At this time, this tool does not check slice
// operations in a way this expansion could help. See
// https://play.golang.org/p/mGqXEp7w4fR for an example.
func expandFacts(f fact) []fact {
	ff := []fact{f}

Loop:
	for {
		switch v := f.value.(type) {
		case *ssa.ChangeInterface:
			f = fact{v.X, f.nilness}
			ff = append(ff, f)
		default:
			break Loop
		}
	}

	return ff
}

type facts []fact

func (ff facts) negate() facts {
	nn := make([]fact, len(ff))
	for i, f := range ff {
		nn[i] = f.negate()
	}
	return nn
}

type validator struct {
	pass    *analysis.Pass
	targets []*target
}

type target struct {
	fn     *types.Func
	argPos int
}

// Target represents a function or a method to be checked by nilnop.
type Target struct {
	// Package path of the function or method.
	PkgPath string
	// Name of the function or method.
	FuncName string
	// Position of an argument which should not be nil.
	// ArgPos is 0-indexed.
	ArgPos int
}

func newValidator(pass *analysis.Pass, ts []Target) (*validator, error) {
	targets := make([]*target, 0, len(ts))
	for _, t := range ts {
		fn, err := funcObjectOf(pass, t)
		if err != nil {
			return nil, err
		}
		if fn == nil {
			continue
		}
		tgt := &target{
			fn:     fn,
			argPos: t.ArgPos,
		}
		targets = append(targets, tgt)
	}
	return &validator{
		pass:    pass,
		targets: targets,
	}, nil
}

func (v *validator) validate(stack []fact, instr ssa.CallInstruction) {
	defer func() {
		recover()
	}()
	for _, t := range v.targets {
		if t.fn == instr.Common().Value.(*ssa.Function).Object() {
			fn := instr.Common().Value.(*ssa.Function)
			if fn.Signature.Recv() == nil {
				arg := instr.Common().Args[t.argPos]
				if nilnessOf(stack, arg) == isnil {
					v.pass.Reportf(instr.Pos(), "nil is passed to %s", fn.Name())
				}
			} else {
				arg := instr.Common().Args[t.argPos+1]
				if nilnessOf(stack, arg) == isnil {
					v.pass.Reportf(instr.Pos(), "nil is passed to %s", fn.Name())
				}
			}
			return
		}
	}
}

func funcObjectOf(pass *analysis.Pass, tgt Target) (*types.Func, error) {
	// function
	if !strings.Contains(tgt.FuncName, ".") {
		obj := analysisutil.ObjectOf(pass, tgt.PkgPath, tgt.FuncName)
		if obj == nil {
			// not found is ok because func need not to be called.
			return nil, nil
		}
		ft, ok := obj.(*types.Func)
		if !ok {
			return nil, newErrNotFunc(tgt.PkgPath, tgt.FuncName)
		}
		return ft, nil
	}
	tt := strings.Split(tgt.FuncName, ".")
	if len(tt) != 2 {
		return nil, newErrInvalidFuncName(tgt.FuncName)
	}
	// method
	recv := tt[0]
	method := tt[1]
	recvType := analysisutil.TypeOf(pass, tgt.PkgPath, recv)
	if recvType == nil {
		// not found is ok because method need not to be called.
		return nil, nil
	}
	m := analysisutil.MethodOf(recvType, method)
	if m == nil {
		// not found is ok because method need not to be called.
		return nil, nil
	}
	return m, nil
}

type errInvalidFuncName struct {
	FuncName string
}

func newErrInvalidFuncName(funcName string) errInvalidFuncName {
	return errInvalidFuncName{
		FuncName: funcName,
	}
}

func (e errInvalidFuncName) Error() string {
	return fmt.Sprintf("invalid FuncName %s", e.FuncName)
}

type errNotFunc struct {
	PkgPath  string
	FuncName string
}

func newErrNotFunc(pkgPath, funcName string) errNotFunc {
	return errNotFunc{
		PkgPath:  pkgPath,
		FuncName: funcName,
	}
}

func (e errNotFunc) Error() string {
	return fmt.Sprintf("%s.%s is not a function.", e.PkgPath, e.FuncName)
}
