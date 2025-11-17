package states

type SelectedSource int

const (
	SelectedSource_Default SelectedSource = iota
	SelectedSource_Search
	SelectedSource_NavigateByKey
)

type SelectedEntryMode int

const (
	SelectedEntryMode_Default = iota
	SelectedEntryMode_Editing
	SelectedEntryMode_ShowActions
	SelectedEntryMode_DeleteConfirm
	SelectedEntryMode_AddingChild
)
