package main

import (
	"embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"time"
)

const (
	ScreenWidth  = 800
	ScreenHeight = 600
)

//go:embed assets/*
var assets embed.FS

var ScoreFont = mustLoadFont("assets/font.ttf")

func mustLoadFont(name string) font.Face {
	f, err := assets.ReadFile(name)
	if err != nil {
		panic(err)
	}

	tt, err := opentype.Parse(f)
	if err != nil {
		panic(err)
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    48,
		DPI:     72,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		panic(err)
	}

	return face
}

var PlayerSprite = mustLoadImage("assets/player.png")

func mustLoadImage(name string) *ebiten.Image {
	f, err := assets.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	return ebiten.NewImageFromImage(img)
}

func mustLoadImages(dir string) []*ebiten.Image {
	entries, err := assets.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	var images []*ebiten.Image
	for _, entry := range entries {
		images = append(images, mustLoadImage(dir+"/"+entry.Name()))
	}

	return images
}

type Vector struct {
	X float64
	Y float64
}

func (v *Vector) Normalize() Vector {
	mag := math.Sqrt(v.X*v.X + v.Y*v.Y)
	return Vector{
		X: v.X / mag,
		Y: v.Y / mag,
	}
}

type Timer struct {
	currentTicks int
	targetTicks  int
}

func NewTimer(d time.Duration) *Timer {
	return &Timer{
		currentTicks: 0,
		targetTicks:  int(d.Milliseconds()) * ebiten.TPS() / 1000,
	}
}

func (t *Timer) Update() {
	if t.currentTicks < t.targetTicks {
		t.currentTicks++
	}
}

func (t *Timer) IsReady() bool {
	return t.currentTicks >= t.targetTicks
}

func (t *Timer) Reset() {
	t.currentTicks = 0
}

var MeteorSprites = mustLoadImages("assets/meteors")

type Meteor struct {
	Position      Vector
	Movement      Vector
	Rotation      float64
	rotationSpeed float64
	Sprite        *ebiten.Image
}

func NewMeteor() *Meteor {
	// Figure out the target position — the screen center, in this case
	target := Vector{
		X: ScreenWidth / 2,
		Y: ScreenHeight / 2,
	}

	// The distance from the center the meteor should spawn at — half the width
	r := ScreenWidth / 2.0

	// Pick a random angle — 2π is 360° — so this returns 0° to 360°
	angle := rand.Float64() * 2 * math.Pi

	// Figure out the spawn position by moving r pixels from the target at the chosen angle
	pos := Vector{
		X: target.X + math.Cos(angle)*r,
		Y: target.Y + math.Sin(angle)*r,
	}

	// Randomized velocity
	velocity := 0.25 + rand.Float64()*1.5

	// Direction is the target minus the current position
	direction := Vector{
		X: target.X - pos.X,
		Y: target.Y - pos.Y,
	}

	// Normalize the vector — get just the direction without the length
	normalizedDirection := direction.Normalize()

	// Multiply the direction by velocity
	movement := Vector{
		X: normalizedDirection.X * velocity,
		Y: normalizedDirection.Y * velocity,
	}

	sprite := MeteorSprites[rand.Intn(len(MeteorSprites))]

	return &Meteor{
		Position:      pos,
		Movement:      movement,
		Rotation:      0,
		rotationSpeed: -0.02 + rand.Float64()*0.04,
		Sprite:        sprite,
	}
}

func (m *Meteor) Collider() Rect {
	bounds := m.Sprite.Bounds()

	return Rect{
		X:      m.Position.X,
		Y:      m.Position.Y,
		Width:  float64(bounds.Dx()),
		Height: float64(bounds.Dy()),
	}
}

func (m *Meteor) Update() error {
	m.Position.X += m.Movement.X
	m.Position.Y += m.Movement.Y
	m.Rotation += m.rotationSpeed
	return nil
}

func (m *Meteor) Draw(screen *ebiten.Image) {
	bounds := m.Sprite.Bounds()
	halfW := float64(bounds.Dx()) / 2
	halfH := float64(bounds.Dy()) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-halfW, -halfH)
	op.GeoM.Rotate(m.Rotation)
	op.GeoM.Translate(halfW, halfH)

	op.GeoM.Translate(m.Position.X, m.Position.Y)

	screen.DrawImage(m.Sprite, op)
}

var BulletSprite = mustLoadImage("assets/lasers/laserBlue01.png")

type Bullet struct {
	position Vector
	rotation float64
	sprite   *ebiten.Image
}

func NewBullet(pos Vector, rot float64) *Bullet {
	sprite := BulletSprite

	bounds := sprite.Bounds()
	halfW := float64(bounds.Dx()) / 2
	halfH := float64(bounds.Dy()) / 2

	pos.X -= halfW
	pos.Y -= halfH

	return &Bullet{
		position: pos,
		rotation: rot,
		sprite:   sprite,
	}
}

func (b *Bullet) Collider() Rect {
	bounds := b.sprite.Bounds()

	return Rect{
		X:      b.position.X,
		Y:      b.position.Y,
		Width:  float64(bounds.Dx()),
		Height: float64(bounds.Dy()),
	}
}

func (b *Bullet) Update() error {
	speed := 350 / float64(ebiten.TPS())

	b.position.X += math.Sin(b.rotation) * speed
	b.position.Y += math.Cos(b.rotation) * -speed

	return nil
}

func (b *Bullet) Draw(screen *ebiten.Image) {
	bounds := b.sprite.Bounds()
	halfW := float64(bounds.Dx()) / 2
	halfH := float64(bounds.Dy()) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-halfW, -halfH)
	op.GeoM.Rotate(b.rotation)
	op.GeoM.Translate(halfW, halfH)

	op.GeoM.Translate(b.position.X, b.position.Y)

	screen.DrawImage(b.sprite, op)
}

type BulletAdder interface {
	AddBullet(b *Bullet)
}

type Player struct {
	position      Vector
	rotation      float64
	shootCooldown *Timer
	sprite        *ebiten.Image
	bulletAdder   BulletAdder
}

func NewPlayer(bulletAdder BulletAdder) *Player {
	sprite := PlayerSprite

	bounds := sprite.Bounds()
	halfW := float64(bounds.Dx()) / 2
	halfH := float64(bounds.Dy()) / 2

	pos := Vector{
		X: ScreenWidth/2 - halfW,
		Y: ScreenHeight/2 - halfH,
	}

	return &Player{
		position:      pos,
		rotation:      0,
		shootCooldown: NewTimer(1 * time.Second),
		sprite:        sprite,
		bulletAdder:   bulletAdder,
	}
}

func (p *Player) Collider() Rect {
	bounds := p.sprite.Bounds()

	return Rect{
		X:      p.position.X,
		Y:      p.position.Y,
		Width:  float64(bounds.Dx()),
		Height: float64(bounds.Dy()),
	}
}

func (p *Player) Update() error {
	speed := math.Pi / float64(ebiten.TPS())

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		p.rotation -= speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		p.rotation += speed
	}

	p.shootCooldown.Update()
	if p.shootCooldown.IsReady() && ebiten.IsKeyPressed(ebiten.KeySpace) {
		p.shootCooldown.Reset()

		bulletSpawnOffset := 50.0

		bounds := p.sprite.Bounds()
		halfW := float64(bounds.Dx()) / 2
		halfH := float64(bounds.Dy()) / 2

		spawnPos := Vector{
			p.position.X + halfW + math.Sin(p.rotation)*bulletSpawnOffset,
			p.position.Y + halfH + math.Cos(p.rotation)*-bulletSpawnOffset,
		}

		b := NewBullet(spawnPos, p.rotation)

		p.bulletAdder.AddBullet(b)
	}

	return nil
}

func (p *Player) Draw(screen *ebiten.Image) {
	bounds := p.sprite.Bounds()
	halfW := float64(bounds.Dx()) / 2
	halfH := float64(bounds.Dy()) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-halfW, -halfH)
	op.GeoM.Rotate(p.rotation)
	op.GeoM.Translate(halfW, halfH)

	op.GeoM.Translate(p.position.X, p.position.Y)

	screen.DrawImage(p.sprite, op)
}

type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

func NewRect(x, y, width, height float64) Rect {
	return Rect{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}

func (r Rect) MaxX() float64 {
	return r.X + r.Width
}

func (r Rect) MaxY() float64 {
	return r.Y + r.Height
}

func (r Rect) Intersects(other Rect) bool {
	return r.X <= other.MaxX() &&
		other.X <= r.MaxX() &&
		r.Y <= other.MaxY() &&
		other.Y <= r.MaxY()
}

type Game struct {
	player           *Player
	score            int
	meteorSpawnTimer *Timer
	meteors          []*Meteor
	bullets          []*Bullet
}

func (g *Game) AddBullet(b *Bullet) {
	g.bullets = append(g.bullets, b)
}

func (g *Game) Update() error {
	err := g.player.Update()
	if err != nil {
		return err
	}

	g.meteorSpawnTimer.Update()
	if g.meteorSpawnTimer.IsReady() {
		g.meteorSpawnTimer.Reset()

		m := NewMeteor()
		g.meteors = append(g.meteors, m)
	}

	for _, m := range g.meteors {
		err = m.Update()
		if err != nil {
			return err
		}
	}

	for _, b := range g.bullets {
		err = b.Update()
		if err != nil {
			return err
		}
	}

	for i, m := range g.meteors {
		for j, b := range g.bullets {
			if m.Collider().Intersects(b.Collider()) {
				// A meteor collided with a bullet
				g.meteors = append(g.meteors[:i], g.meteors[i+1:]...)
				g.bullets = append(g.bullets[:j], g.bullets[j+1:]...)

				// Increase the score
				g.score++
			}
		}
	}

	for _, m := range g.meteors {
		if m.Collider().Intersects(g.player.Collider()) {
			// A meteor collided with the player
			g.Reset()
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.player.Draw(screen)

	for _, m := range g.meteors {
		m.Draw(screen)
	}

	for _, b := range g.bullets {
		b.Draw(screen)
	}

	text.Draw(screen, fmt.Sprintf("%06d", g.score), ScoreFont, ScreenWidth/2-100, 50, color.White)
}

func (g *Game) Layout(outsideWith, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}

func (g *Game) Reset() {
	g.player = NewPlayer(g)
	g.meteors = nil
	g.bullets = nil
	g.score = 0
}

func main() {

	g := &Game{
		meteorSpawnTimer: NewTimer(5 * time.Second),
		meteors:          nil,
		bullets:          nil,
	}

	g.player = NewPlayer(g)

	err := ebiten.RunGame(g)
	if err != nil {
		log.Fatal(err)
	}
}
