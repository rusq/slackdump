// Code generated by "stringer -type=Focus"; DO NOT EDIT.

package datepicker

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[FocusNone-0]
	_ = x[FocusHeaderMonth-1]
	_ = x[FocusHeaderYear-2]
	_ = x[FocusCalendar-3]
}

const _Focus_name = "FocusNoneFocusHeaderMonthFocusHeaderYearFocusCalendar"

var _Focus_index = [...]uint8{0, 9, 25, 40, 53}

func (i Focus) String() string {
	if i < 0 || i >= Focus(len(_Focus_index)-1) {
		return "Focus(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Focus_name[_Focus_index[i]:_Focus_index[i+1]]
}
