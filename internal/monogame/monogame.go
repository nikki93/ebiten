// Copyright 2020 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build js

package monogame

import (
	"reflect"
	"runtime"
	"syscall/js"
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
)

// TODO: This implementation depends on some C# files that are not uploaded yet.
// Create 'ebitenmonogame' command to generate C# project for the MonoGame.

// namespace is C# namespace.
//
// This is overwritten by -ldflags='-X github.com/hajimehoshi/ebiten/internal/monogame.namespace=NAMESPACE'.
var namespace = "Go2DotNet.Example.Ebiten"

type UpdateDrawer interface {
	Update() error
	Draw() error
}

type Game struct {
	binding js.Value
	update  js.Func
	draw    js.Func
}

func NewGame(ud UpdateDrawer) *Game {
	update := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return ud.Update()
	})

	draw := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return ud.Draw()
	})

	v := js.Global().Get(".net").Get(namespace+".GameGoBinding").New(update, draw)
	g := &Game{
		binding: v,
		update:  update,
		draw:    draw,
	}
	runtime.SetFinalizer(g, (*Game).Dispose)
	return g
}

func (g *Game) Dispose() {
	runtime.SetFinalizer(g, nil)
	g.update.Release()
	g.draw.Release()
}

func (g *Game) Run() {
	g.binding.Call("Run")
}

func (g *Game) NewRenderTarget2D(width, height int) *RenderTarget2D {
	v := g.binding.Call("NewRenderTarget2D", width, height)
	r := &RenderTarget2D{
		v:       v,
		binding: g.binding,
	}
	runtime.SetFinalizer(r, (*RenderTarget2D).Dispose)
	return r
}

func (g *Game) SetVertices(vertices []float32, indices []uint16) {
	var vs, is js.Value
	{
		h := (*reflect.SliceHeader)(unsafe.Pointer(&vertices))
		h.Len *= 4
		h.Cap *= 4
		bs := *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(vertices)
		vs = js.Global().Get("Uint8Array").New(len(bs))
		js.CopyBytesToJS(vs, bs)
	}
	{
		h := (*reflect.SliceHeader)(unsafe.Pointer(&indices))
		h.Len *= 2
		h.Cap *= 2
		bs := *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(indices)
		is = js.Global().Get("Uint8Array").New(len(bs))
		js.CopyBytesToJS(is, bs)
	}
	g.binding.Call("SetVertices", vs, is)
}

func (g *Game) Draw(indexLen int, indexOffset int, mode driver.CompositeMode, colorM *affine.ColorM, filter driver.Filter, address driver.Address) {
	src, dst := mode.Operations()
	g.binding.Call("Draw", indexLen, indexOffset, int(src), int(dst))
}

func (g *Game) ResetDestination(viewportWidth, viewportHeight int) {
	g.binding.Call("SetDestination", nil, viewportWidth, viewportHeight)
}

func (g *Game) IsKeyPressed(key driver.Key) bool {
	// Pass a string of the key since both driver.Key value and XNA's key value are not reliable.
	return g.binding.Call("IsKeyPressed", key.String()).Bool()
}

type RenderTarget2D struct {
	v       js.Value
	binding js.Value
}

func (r *RenderTarget2D) Dispose() {
	runtime.SetFinalizer(r, nil)
	r.binding.Call("Dispose", r.v)
}

func (r *RenderTarget2D) Pixels(width, height int) ([]byte, error) {
	v := r.binding.Call("Pixels", r.v, width, height)
	bs := make([]byte, v.Length())
	js.CopyBytesToGo(bs, v)
	return bs, nil
}

func (r *RenderTarget2D) ReplacePixels(args []*driver.ReplacePixelsArgs) {
	for _, a := range args {
		arr := js.Global().Get("Uint8Array").New(len(a.Pixels))
		js.CopyBytesToJS(arr, a.Pixels)
		r.binding.Call("ReplacePixels", r.v, arr, a.X, a.Y, a.Width, a.Height)
	}
}

func (r *RenderTarget2D) SetAsDestination(viewportWidth, viewportHeight int) {
	r.binding.Call("SetDestination", r.v, viewportWidth, viewportHeight)
}

func (r *RenderTarget2D) SetAsSource() {
	r.binding.Call("SetSource", r.v)
}

func (r *RenderTarget2D) IsScreen() bool {
	return false
}
