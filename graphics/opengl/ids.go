package opengl

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/matrix"
	"github.com/hajimehoshi/ebiten/graphics/opengl/internal/shader"
	"image"
	"math"
	"sync"
)

func glMatrix(matrix [4][4]float64) [16]float32 {
	result := [16]float32{}
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			result[i+j*4] = float32(matrix[i][j])
		}
	}
	return result
}

type ids struct {
	textures              map[graphics.TextureID]*Texture
	renderTargets         map[graphics.RenderTargetID]*RenderTarget
	renderTargetToTexture map[graphics.RenderTargetID]graphics.TextureID
	lastId                int
	currentRenderTargetId graphics.RenderTargetID
	sync.RWMutex
}

var idsInstance = &ids{
	textures:              map[graphics.TextureID]*Texture{},
	renderTargets:         map[graphics.RenderTargetID]*RenderTarget{},
	renderTargetToTexture: map[graphics.RenderTargetID]graphics.TextureID{},
	currentRenderTargetId: -1,
}

func CreateRenderTarget(
	width, height int,
	filter graphics.Filter) (graphics.RenderTargetID, error) {
	return idsInstance.createRenderTarget(width, height, filter)
}

func CreateTexture(
	img image.Image,
	filter graphics.Filter) (graphics.TextureID, error) {
	return idsInstance.createTexture(img, filter)
}

func (i *ids) textureAt(id graphics.TextureID) *Texture {
	i.RLock()
	defer i.RUnlock()
	return i.textures[id]
}

func (i *ids) renderTargetAt(id graphics.RenderTargetID) *RenderTarget {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargets[id]
}

func (i *ids) toTexture(id graphics.RenderTargetID) graphics.TextureID {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargetToTexture[id]
}

func (i *ids) createTexture(img image.Image, filter graphics.Filter) (graphics.TextureID, error) {
	texture, err := createTextureFromImage(img, filter)
	if err != nil {
		return 0, err
	}

	i.Lock()
	defer i.Unlock()
	i.lastId++
	textureId := graphics.TextureID(i.lastId)
	i.textures[textureId] = texture
	return textureId, nil
}

func (i *ids) createRenderTarget(width, height int, filter graphics.Filter) (graphics.RenderTargetID, error) {
	texture, err := createTexture(width, height, filter)
	if err != nil {
		return 0, err
	}
	framebuffer := createFramebuffer(gl.Texture(texture.native))
	// The current binded framebuffer can be changed.
	i.currentRenderTargetId = -1
	renderTarget := &RenderTarget{
		framebuffer: framebuffer,
		width:       texture.width,
		height:      texture.height,
	}

	i.Lock()
	defer i.Unlock()
	i.lastId++
	textureId := graphics.TextureID(i.lastId)
	i.lastId++
	renderTargetId := graphics.RenderTargetID(i.lastId)

	i.textures[textureId] = texture
	i.renderTargets[renderTargetId] = renderTarget
	i.renderTargetToTexture[renderTargetId] = textureId

	return renderTargetId, nil
}

// NOTE: renderTarget can't be used as a texture.
func (i *ids) addRenderTarget(renderTarget *RenderTarget) graphics.RenderTargetID {
	i.Lock()
	defer i.Unlock()
	i.lastId++
	id := graphics.RenderTargetID(i.lastId)
	i.renderTargets[id] = renderTarget

	return id
}

func (i *ids) deleteRenderTarget(id graphics.RenderTargetID) {
	i.Lock()
	defer i.Unlock()

	renderTarget := i.renderTargets[id]
	textureId := i.renderTargetToTexture[id]
	texture := i.textures[textureId]

	renderTarget.dispose()
	texture.dispose()

	delete(i.renderTargets, id)
	delete(i.renderTargetToTexture, id)
	delete(i.textures, textureId)
}

func (i *ids) fillRenderTarget(id graphics.RenderTargetID, r, g, b uint8) {
	i.setViewportIfNeeded(id)
	const max = float64(math.MaxUint8)
	gl.ClearColor(gl.GLclampf(float64(r)/max), gl.GLclampf(float64(g)/max), gl.GLclampf(float64(b)/max), 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func (i *ids) drawTexture(
	target graphics.RenderTargetID,
	id graphics.TextureID,
	parts []graphics.TexturePart,
	geo matrix.Geometry,
	color matrix.Color) {
	texture := i.textureAt(id)
	i.setViewportIfNeeded(target)
	r := i.renderTargetAt(target)
	projectionMatrix := r.projectionMatrix()
	quads := graphics.TextureQuads(parts, texture.width, texture.height)
	shader.DrawTexture(texture.native, glMatrix(projectionMatrix), quads, geo, color)
}

func (i *ids) setViewportIfNeeded(id graphics.RenderTargetID) {
	r := i.renderTargetAt(id)
	if i.currentRenderTargetId != id {
		r.setAsViewport()
		i.currentRenderTargetId = id
	}
}
