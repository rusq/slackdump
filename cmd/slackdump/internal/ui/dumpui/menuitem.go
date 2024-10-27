package dumpui

// MenuItem is an item in a menu.
type MenuItem struct {
	// ID is an arbitrary ID, up to caller.
	ID string
	// Separator is a flag that determines whether the item is a separator or not.
	Separator bool
	// Name is the name of the Item, will be displayed in the menu.
	Name string
	// Help is the help text for the item, that will be shown when the
	// item is highlighted.
	Help string
	// Model is any model that should be displayed when the item is selected,
	// or executed when the user presses enter.
	Model FocusModel
	// IsDisabled determines whether the item is disabled or not. It should
	// complete in reasonable time, as it is called on every render.
	IsDisabled func() bool // when to enable the item
}
