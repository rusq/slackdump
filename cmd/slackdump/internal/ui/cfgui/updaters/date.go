package updaters

import (
	"time"

	datepicker "github.com/ethanefung/bubble-datepicker"
)

type DateModel struct {
	Value *time.Time
	m     datepicker.Model
}
