// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittymodule

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

// A module can check what it reads, when an owner asks why a disc won't play. A helper's check
// reports as it goes, through notifications, and a server shows it with its other checks. What
// it says is words about the disc's structure: a report holds no disc content.
const (
	NotifyDiagStep     = "diag.step"     // DiagStep: what it's doing now
	NotifyDiagProgress = "diag.progress" // a number from 0 to 1
	NotifyDiagLabel    = "diag.label"    // the disc's name, if it has one
	NotifyDiagFinding  = "diag.finding"  // DiagFinding
)

// How bad a finding is.
const (
	DiagOK      = "ok"
	DiagInfo    = "info"
	DiagWarn    = "warn"
	DiagProblem = "problem"
)

// DiagStep is what a check is doing, and how far along it is, from 0 to 1.
type DiagStep struct {
	Phase    string  `json:"phase"`
	Fraction float64 `json:"fraction"`
}

// DiagFinding is one thing a check found.
type DiagFinding struct {
	Severity string   `json:"severity"`
	Title    string   `json:"title"`
	What     string   `json:"what,omitempty"`
	Fix      string   `json:"fix,omitempty"`
	Details  []string `json:"details,omitempty"`
	// Names are file names (base names only). They come out of a report downloaded with
	// names hidden, so nothing else here may hold one.
	Names []string `json:"names,omitempty"`
	// More counts what Names and Details left out.
	More int `json:"more,omitempty"`
}

// DiagJob is a check in a helper, reporting to the server.
type DiagJob struct{}

// Step says what it's doing now.
func (j *DiagJob) Step(phase string, fraction float64) {
	_ = Notify(NotifyDiagStep, DiagStep{Phase: phase, Fraction: fraction})
}

// Progress says how far along it is.
func (j *DiagJob) Progress(fraction float64) { _ = Notify(NotifyDiagProgress, fraction) }

// SetLabel gives the disc's name.
func (j *DiagJob) SetLabel(label string) { _ = Notify(NotifyDiagLabel, label) }

// Add reports something found.
func (j *DiagJob) Add(f DiagFinding) { _ = Notify(NotifyDiagFinding, f) }

// Rest gives way to others between pieces of work: a helper runs at a low priority already, so
// this only says whether to carry on.
func (j *DiagJob) Rest(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }

// Engine runs a part of a check. A crash there is a result, not a disaster: it's turned into
// a finding, and the rest of the check carries on. ok is false after a crash.
func (j *DiagJob) Engine(what string, fn func()) (ok bool) {
	defer func() {
		if p := recover(); p != nil {
			j.Add(DiagFinding{
				Severity: DiagProblem,
				Title:    "The engine crashed on this disc",
				What:     "Something went wrong inside the module while " + what + ". That's a bug in the module, and is its fault rather than the disc's or yours.",
				Fix:      "The rest of the check carried on. Please send this report (it holds no disc content) with a bug report.",
				Details:  []string{fmt.Sprint(p), string(debug.Stack())},
			})
			ok = false
		}
	}()
	fn()
	return true
}
