// Code generated by "stringer -type=ResultType -trimprefix=RT"; DO NOT EDIT.

package slackdump

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[RTMain-0]
	_ = x[RTChannel-1]
	_ = x[RTThread-2]
}

const _ResultType_name = "MainChannelThread"

var _ResultType_index = [...]uint8{0, 4, 11, 17}

func (i ResultType) String() string {
	if i < 0 || i >= ResultType(len(_ResultType_index)-1) {
		return "ResultType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ResultType_name[_ResultType_index[i]:_ResultType_index[i+1]]
}