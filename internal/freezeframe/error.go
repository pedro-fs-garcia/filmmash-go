package freezeframe

import "errors"

var ErrFilmMustBeSet = errors.New("film must be set")

var ErrInvalidReel = errors.New("invalid reel")
var ErrInvalidFrame = errors.New("invalid frame")
var ErrInvalidAlternative = errors.New("invalid alternative")
var ErrInvalidReelFrame = errors.New("invalid reel_frame")
var ErrInvalidGame = errors.New("invalid game")
