package menu

// MenuItem is an item in a menu.
type MenuItem struct {
	// ID is an arbitrary ID, up to caller.
	ID string
	// Separator is a flag that determines whether the item is a separator or
	// not.
	Separator bool
	// Name is the name of the Item, will be displayed in the menu.
	Name string
	// Help is the help text for the item, that will be shown when the
	// item is highlighted.
	Help string
	// Model is any model that should be displayed when the item is selected,
	// or executed when the user presses enter.
	Model FocusModel
	// Preview suggests that the Model should attempt to show the preview
	// of this item.
	Preview bool
	// Validate determines whether the item is disabled or not. It should
	// complete in reasonable time, as it is called on every render.  The
	// return error is used in the description for the item.
	Validate func() error // when to enable the item
}

func (m MenuItem) IsDisabled() bool {
	return m.Validate != nil && m.Validate() != nil
}

func (m MenuItem) DisabledReason() string {
	if m.Validate != nil {
		if err := m.Validate(); err != nil {
			return err.Error()
		}
	}
	return ""
}
