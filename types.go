package main

import "sync"

func btoi(a bool) int {
	if a {
		return 1
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(min int, i int, max int) int {
	if i < min {
		return min
	} else if i > max {
		return max
	} else {
		return i
	}
}

type Entity struct {
	Valid bool
	Pos   Vector
	Vel   Vector
	Size  Size
	Color RGBA
	Image int
}

type Vector struct {
	X int32
	Y int32
}

type Size struct {
	W int32
	H int32
}

type RGBA struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

type GameState struct {
	Time      int32
	FrameTime int32
	Entities  [1024]Entity
	Camera    Camera
	LocalEnt  *Entity
}

type EventType int

const (
	EV_KEY EventType = 1 + iota
	EV_MOUSEMOVE
	EV_MOUSECLICK
	EV_MOUSEWHEEL
)

type Event struct {
	Type     EventType
	Down     bool
	Position Vector
	EvData1  int
	EvData2  int
}

type Scene interface {
	Load(sceneCh SceneChannels)
	Unload()
}

type UserCommand struct {
	Up    int
	Down  int
	Left  int
	Right int
}

type SceneChannels struct {
	RCmd      chan *RenderCommandList
	Ev        chan Event
	Eng       chan EngineCommand
	Err       chan error
	stateLock sync.Mutex
}

type ECmd int

const (
	EC_LOADIMAGE ECmd = 1 + iota
)

type EngineCommand struct {
	Id      ECmd
	Success bool
	Data    interface{}
}

type Image struct {
	Id int
	W  int32
	H  int32
}

type RCmd int

const (
	RC_RECT RCmd = 1 + iota
	RC_PIC
	RC_TEXT
)

type RenderCommandList struct {
	NumCommands int32
	Commands    [2048]RenderCommand
}

type RenderCommand struct {
	Id        RCmd
	Pos       Vector
	Size      Size
	ImageId   int32
	ImgPos    Vector
	ImgSize   Size
	BackColor RGBA
}

type Camera struct {
	Left   int
	Right  int
	Top    int
	Bottom int
	Bounds Size
	Size   Size
}

func (s *Camera) Set(pos Vector) {
	s.Left = clamp(0, int(pos.X), int(s.Bounds.W-s.Size.W))
	s.Top = clamp(0, int(pos.Y), int(s.Bounds.H-s.Size.H))
	s.Right = s.Left + int(s.Size.W)
	s.Bottom = s.Top + int(s.Size.H)
}

func (s *Camera) SetSize(sz Size) {
	s.Size = sz
	s.Set(Vector{int32(s.Left), int32(s.Top)})
}

func (s *Camera) SetBounds(b Size) {
	s.Bounds = b
	s.Set(Vector{int32(s.Left), int32(s.Top)})
}
