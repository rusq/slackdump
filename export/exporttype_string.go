// Code generated by "stringer -type=ExportType -linecomment"; DO NOT EDIT.

package export

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[TStandard-0]
	_ = x[TMattermost-1]
}

const _ExportType_name = "StandardMattermost"

var _ExportType_index = [...]uint8{0, 8, 18}

func (i ExportType) String() string {
	if i >= ExportType(len(_ExportType_index)-1) {
		return "ExportType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ExportType_name[_ExportType_index[i]:_ExportType_index[i+1]]
}