package tray

type IconType int

const (
	IconIdle   IconType = iota
	IconBlue
	IconYellow
	IconRed
)

type IconState struct {
	newSession bool
	expiring   bool
	expired    bool
}

func NewIconState() *IconState {
	return &IconState{}
}

func (s *IconState) SetNewSession(v bool) { s.newSession = v }
func (s *IconState) SetExpiring(v bool)   { s.expiring = v }
func (s *IconState) SetExpired(v bool)    { s.expired = v }

func (s *IconState) Reset() {
	s.newSession = false
	s.expiring = false
	s.expired = false
}

func (s *IconState) Current() IconType {
	if s.expired {
		return IconRed
	}
	if s.expiring {
		return IconYellow
	}
	if s.newSession {
		return IconBlue
	}
	return IconIdle
}
