package main

import (
	"container/ring"
	"sync"

	"github.com/mgutz/ansi"
)

type colorPicker struct {
	colors []string
	it     *ring.Ring
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
	cp.it = ring.New(len(cp.colors))
	for i := 0; i < cp.it.Len(); i++ {
		cp.it.Value = cp.colors[i]
		cp.it = cp.it.Next()
	}
	return cp
}

func (cp *colorPicker) next() string {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	color, _ := cp.it.Value.(string)
	cp.it = cp.it.Next()
	return color
}

func (cp *colorPicker) reset() string {
	return ansi.ColorCode("reset")
}
