package main

import (
	"sync"

	"github.com/mgutz/ansi"
)

type colorPicker struct {
	colors []string
	i      int
	mutex  sync.Mutex
}

func newColorPicker() *colorPicker {
	cp := &colorPicker{}
	colors := []string{"cyan", "yellow", "green", "magenta", "blue", "red"}
	cp.colors = []string{}
	for _, c := range colors {
		cp.colors = append(cp.colors, ansi.ColorCode(c))
	}
	cp.mutex = sync.Mutex{}
	cp.i = 0
	return cp
}

func (cp *colorPicker) next() string {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	if cp.i >= len(cp.colors) {
		cp.i = 0
	}
	color := cp.colors[cp.i]
	cp.i += 1
	return color
}

func (cp *colorPicker) reset() string {
	return ansi.ColorCode("reset")
}
