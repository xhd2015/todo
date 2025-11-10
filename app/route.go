package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/xhd2015/go-dom-tui/colors"
	"github.com/xhd2015/go-dom-tui/dom"
	"github.com/xhd2015/go-dom-tui/styles"
	"github.com/xhd2015/todo/app/happening_list"
	"github.com/xhd2015/todo/app/help"
	"github.com/xhd2015/todo/app/human_state"
	"github.com/xhd2015/todo/app/learning"
	"github.com/xhd2015/todo/component/text"
	"github.com/xhd2015/todo/log"
	"github.com/xhd2015/todo/models"
)

type RouteType int

const (
	RouteType_Main RouteType = iota
	RouteType_Detail
	RouteType_Config
	RouteType_HappeningList
	RouteType_HumanState
	RouteType_Help
	RouteType_Learning
	RouteType_Reading
)

type Routes []Route

type Route struct {
	Type              RouteType
	MainPage          *MainPageState
	DetailPage        *DetailPageState
	ConfigPage        *ConfigPageState
	HappeningListPage *HappeningListPageState
	HumanStatePage    *HumanStatePageState
	HelpPage          *HelpPageState
	LearningPage      *LearningPageState
	ReadingPage       *ReadingPageState
}

func (routes *Routes) Push(route Route) {
	*routes = append(*routes, route)
}

func (routes *Routes) Pop() {
	*routes = (*routes)[:len(*routes)-1]
}

func (routes *Routes) Last() Route {
	return (*routes)[len(*routes)-1]
}

type MainPageState struct {
	Entries []TreeEntry
}

type DetailPageState struct {
	EntryID int64
}

type ConfigPhase int

const (
	ConfigPhase_PickingStorageType ConfigPhase = iota
	ConfigPhase_PickingStorageDetail
)

type StorageType int

const (
	StorageType_LocalFile StorageType = iota
	StorageType_LocalSqlite
	StorageType_Server
)

type ConfigPageState struct {
	ConfigPhase ConfigPhase

	SelectedStorageType StorageType
	PickingStorageType  StorageType

	ServerAddr      models.InputState
	ServerAuthToken models.InputState

	ConfirmButtonFocused bool
	CancelButtonFocused  bool
}

type HappeningListPageState struct {
	// This can be empty since happening state is now in main State
}

type HumanStatePageState struct {
	// This can be empty since human state is now in main State
}

type HelpPageState struct {
	ScrollOffset int // Current scroll position (line offset from top)
}

type LearningPageState struct {
	// This can be empty since learning state is now in main State
}

type ReadingPageState struct {
	MaterialID int64
}

func DetailRoute(entryID int64) Route {
	return Route{
		Type: RouteType_Detail,
		DetailPage: &DetailPageState{
			EntryID: entryID,
		},
	}
}

func ConfigRoute(state ConfigPageState) Route {
	return Route{
		Type:       RouteType_Config,
		ConfigPage: &state,
	}
}

func HappeningListRoute() Route {
	return Route{
		Type:              RouteType_HappeningList,
		HappeningListPage: &HappeningListPageState{},
	}
}

func HumanStateRoute() Route {
	return Route{
		Type:           RouteType_HumanState,
		HumanStatePage: &HumanStatePageState{},
	}
}

func HelpRoute() Route {
	return Route{
		Type:     RouteType_Help,
		HelpPage: &HelpPageState{},
	}
}

func LearningRoute() Route {
	return Route{
		Type:         RouteType_Learning,
		LearningPage: &LearningPageState{},
	}
}

func ReadingRoute(materialID int64) Route {
	return Route{
		Type: RouteType_Reading,
		ReadingPage: &ReadingPageState{
			MaterialID: materialID,
		},
	}
}

// HappeningListPage renders the happening list page
func HappeningListPage(state *State) *dom.Node {
	happeningState := &state.Happening

	if happeningState.Loading {
		return dom.Div(dom.DivProps{},
			dom.Text("Loading happenings..."),
		)
	}

	if happeningState.Error != "" {
		return dom.Div(dom.DivProps{},
			dom.Text("Error loading happenings: "+happeningState.Error),
		)
	}

	return happening_list.HappeningList(happening_list.HappeningListProps{
		Items:         happeningState.Happenings,
		FocusedItemID: happeningState.FocusedItemID,
		OnFocusItem: func(id int64) {
			happeningState.FocusedItemID = id
		},
		OnBlurItem: func(id int64) {
			happeningState.FocusedItemID = 0
		},
		InputState:  &happeningState.Input,
		SubmitState: &happeningState.SubmitState,
		OnNavigateBack: func() {
			// Navigate back to main page by popping the current route
			state.Routes.Pop()
		},
		OnAddHappening: func(text string) {
			// Add new happening using backend API with submission state management
			state.Enqueue(func(ctx context.Context) error {
				return happeningState.SubmitState.Do(ctx, text, func() error {
					if happeningState.AddHappening == nil {
						return fmt.Errorf("AddHappening function not available")
					}

					// Add via backend service
					newHappening, err := happeningState.AddHappening(ctx, text)
					if err != nil {
						return fmt.Errorf("failed to add happening: %w", err)
					}

					// Add to local list for immediate UI update
					happeningState.Happenings = append(happeningState.Happenings, newHappening)
					return nil
				})
			})
		},
		OnReload: func() {
			// Reload happenings by setting loading state and fetching fresh data
			if len(happeningState.Happenings) == 0 {
				happeningState.Loading = true
			}
			happeningState.Error = ""

			state.Enqueue(func(ctx context.Context) error {
				log.Infof(ctx, "Reload happenings")
				if happeningState.LoadHappenings == nil {
					happeningState.Error = "LoadHappenings is not set"
					return nil
				}
				happenings, err := happeningState.LoadHappenings(ctx)
				if err != nil {
					happeningState.Error = err.Error()
					return err
				}
				// Update the state with loaded data
				happeningState.Loading = false
				happeningState.Happenings = happenings
				return nil
			})
		},
		// Edit/Delete functionality
		EditingItemID:       happeningState.EditingItemID,
		EditInputState:      &happeningState.EditInputState,
		DeletingItemID:      happeningState.DeletingItemID,
		DeleteConfirmButton: happeningState.DeleteConfirmButton,
		OnEditItem: func(id int64) {
			// Find the happening to edit
			for _, happening := range happeningState.Happenings {
				if happening.ID == id {
					happeningState.EditingItemID = id
					happeningState.EditInputState.Value = happening.Content
					happeningState.EditInputState.Focused = true
					happeningState.EditInputState.CursorPosition = len(happening.Content)
					break
				}
			}
		},
		OnDeleteItem: func(id int64) {
			happeningState.DeletingItemID = id
			happeningState.DeleteConfirmButton = 0 // Default to Delete button
		},
		OnSaveEdit: func(id int64, content string) {
			// Update happening using backend API
			state.Enqueue(func(ctx context.Context) error {
				if happeningState.UpdateHappening == nil {
					return fmt.Errorf("UpdateHappening function not available")
				}

				// Create update with only the content field
				update := &models.HappeningOptional{
					Content: &content,
				}

				// Update via backend service
				updatedHappening, err := happeningState.UpdateHappening(ctx, id, update)
				if err != nil {
					return fmt.Errorf("update: %w", err)
				}

				// Update local list for immediate UI update
				for i, happening := range happeningState.Happenings {
					if happening.ID == id {
						happeningState.Happenings[i] = updatedHappening
						break
					}
				}

				// Reset edit state
				happeningState.EditingItemID = 0
				happeningState.EditInputState.Reset()
				return nil
			})
		},
		OnCancelEdit: func(e *dom.DOMEvent) {
			happeningState.EditingItemID = 0
			happeningState.EditInputState.Reset()
			if e != nil {
				e.StopPropagation()
			}
		},
		OnConfirmDelete: func(e *dom.DOMEvent, id int64) {
			// Delete happening using backend API
			state.Enqueue(func(ctx context.Context) error {
				if happeningState.DeleteHappening == nil {
					return fmt.Errorf("DeleteHappening function not available")
				}

				// Delete via backend service
				err := happeningState.DeleteHappening(ctx, id)
				if err != nil {
					return fmt.Errorf("failed to delete happening: %w", err)
				}

				// Remove from local list for immediate UI update
				for i, happening := range happeningState.Happenings {
					if happening.ID == id {
						happeningState.Happenings = append(happeningState.Happenings[:i], happeningState.Happenings[i+1:]...)
						break
					}
				}

				// Reset delete state
				happeningState.DeletingItemID = 0
				return nil
			})
		},
		OnCancelDelete: func(e *dom.DOMEvent) {
			happeningState.DeletingItemID = 0
		},
		OnNavigateDeleteConfirm: func(direction int) {
			happeningState.DeleteConfirmButton += direction
			if happeningState.DeleteConfirmButton < 0 {
				happeningState.DeleteConfirmButton = 1
			}
			if happeningState.DeleteConfirmButton > 1 {
				happeningState.DeleteConfirmButton = 0
			}
		},
	})
}

// HumanStatePage renders the human state page
func HumanStatePage(state *State) *dom.Node {
	return human_state.HumanStatePage(
		state.HumanState,
		func(event *dom.DOMEvent) {
			keyEvent := event.KeydownEvent
			if keyEvent != nil {
				if keyEvent.KeyType == dom.KeyTypeEsc {
					state.Routes.Pop()
					return
				}
				return
			}
		},
	)
}

// LearningPage renders the learning materials page
func LearningPage(state *State, width int, height int) *dom.Node {
	learningState := &state.Learning

	if learningState.Loading {
		return dom.Div(dom.DivProps{},
			dom.Text("Loading learning materials..."),
		)
	}

	if learningState.Error != "" {
		return dom.Div(dom.DivProps{},
			dom.Text("Error loading learning materials: "+learningState.Error),
		)
	}

	return learning.LearningMaterialList(learning.LearningMaterialListProps{
		Materials:       learningState.Materials,
		SelectedIndex:   learningState.SelectedMaterialIndex,
		ScrollOffset:    learningState.ScrollOffset,
		ContainerHeight: height,
		ContainerWidth:  width,
		OnNavigateBack: func() {
			state.Routes.Pop()
		},
		OnReload: func() {
			// Reload learning materials by setting loading state and fetching fresh data
			if len(learningState.Materials) == 0 {
				learningState.Loading = true
			}
			learningState.Error = ""

			state.Enqueue(func(ctx context.Context) error {
				log.Infof(ctx, "Reload learning materials")
				if learningState.LoadMaterials == nil {
					learningState.Error = "LoadMaterials is not set"
					return nil
				}
				materials, _, err := learningState.LoadMaterials(ctx, 0, 10)
				if err != nil {
					learningState.Error = err.Error()
					return err
				}
				// Update the state with loaded data
				learningState.Loading = false
				learningState.Materials = materials
				return nil
			})
		},
		OnNavigateUp: func() {
			if learningState.SelectedMaterialIndex > 0 {
				learningState.SelectedMaterialIndex--
				// SliceVertical will automatically adjust scroll position to keep selected item visible
			}
		},
		OnNavigateDown: func() {
			if learningState.SelectedMaterialIndex < len(learningState.Materials)-1 {
				learningState.SelectedMaterialIndex++
				// SliceVertical will automatically adjust scroll position to keep selected item visible
			}
		},
		OnUpdateScrollPos: func(scrollOffset int) {
			// Update scroll offset based on the adjusted beginIndex from SliceVertical
			learningState.ScrollOffset = scrollOffset
		},
		OnOpenMaterial: func(materialID int64) {
			// Initialize reading state
			state.Reading.MaterialID = materialID
			state.Reading.CurrentPage = 0
			state.Reading.ContentCache = make(map[int]string)
			state.Reading.Loading = true
			state.Reading.Error = ""

			// Navigate to reading page
			state.Routes.Push(ReadingRoute(materialID))

			// Load first page
			state.Enqueue(func(ctx context.Context) error {
				return loadPage(ctx, state, 0)
			})

			// Pre-fetch next page
			state.Enqueue(func(ctx context.Context) error {
				return loadPage(ctx, state, 1)
			})
		},
	})
}

// ReadingPage renders the reading page for a material
func ReadingPage(state *State, materialID int64, width int, height int) *dom.Node {
	readingState := &state.Reading

	// Find the material title
	var materialTitle string
	for _, m := range state.Learning.Materials {
		if m.ID == materialID {
			materialTitle = m.Title
			break
		}
	}

	totalPages := 0
	if readingState.TotalBytes > 0 {
		totalPages = (readingState.TotalBytes + PAGE_SIZE - 1) / PAGE_SIZE
	}

	currentContent := readingState.ContentCache[readingState.CurrentPage]

	// Calculate viewport height
	// Reserve space for: title (1), help (1), empty line (1), empty line before pagination (1), pagination (1)
	const HEADER_LINES = 6
	viewportHeight := height - HEADER_LINES
	log.Infof(context.Background(), "viewportHeight: %d", viewportHeight)
	if viewportHeight < 3 {
		viewportHeight = 3 // Minimum viewport height
	}

	return learning.ReadingMaterialPage(learning.ReadingProps{
		MaterialID:       materialID,
		MaterialTitle:    materialTitle,
		CurrentPage:      readingState.CurrentPage,
		TotalPages:       totalPages,
		Content:          currentContent,
		Loading:          readingState.Loading,
		Error:            readingState.Error,
		FocusedWordIndex: readingState.FocusedWordIndex,
		WordPositions:    readingState.WordPositions,
		ScrollOffset:     readingState.ScrollOffset,
		ViewportHeight:   viewportHeight,
		ShowDefinition:   readingState.ShowDefinition,
		DefinitionWord:   readingState.DefinitionWord,
		OnNavigateBack: func() {
			state.Routes.Pop()
		},
		OnNavigateWord: func(delta int) {
			// Navigate by word (left/right)
			readingState.LastKeyWasG = false // Reset 'g' sequence

			if len(readingState.WordPositions) == 0 {
				return
			}

			newIndex := readingState.FocusedWordIndex + delta
			if newIndex < 0 {
				newIndex = 0
			}
			if newIndex >= len(readingState.WordPositions) {
				newIndex = len(readingState.WordPositions) - 1
			}
			readingState.FocusedWordIndex = newIndex

			// Ensure focused word is visible
			ensureWordVisible(readingState, viewportHeight)
		},
		OnNavigateLine: func(delta int) {
			// Navigate by line (up/down)
			readingState.LastKeyWasG = false // Reset 'g' sequence

			if len(readingState.WordPositions) == 0 {
				return
			}

			currentWord := readingState.WordPositions[readingState.FocusedWordIndex]
			currentLine := currentWord.LineIndex

			// Find the target line by searching in the direction of delta
			// If no word exists on the exact target line, find the next available line
			var targetWordIdx int = -1
			bestDistance := -1

			if delta > 0 {
				// Moving down - find first word on a line > currentLine
				for i, wp := range readingState.WordPositions {
					if wp.LineIndex > currentLine {
						if targetWordIdx == -1 || wp.LineIndex < readingState.WordPositions[targetWordIdx].LineIndex {
							// Found a closer line
							targetWordIdx = i
							bestDistance = wp.WordInLine - currentWord.WordInLine
							if bestDistance < 0 {
								bestDistance = -bestDistance
							}
						} else if wp.LineIndex == readingState.WordPositions[targetWordIdx].LineIndex {
							// Same line as current target, check if closer word position
							distance := wp.WordInLine - currentWord.WordInLine
							if distance < 0 {
								distance = -distance
							}
							if distance < bestDistance {
								targetWordIdx = i
								bestDistance = distance
							}
						}
					}
				}
			} else {
				// Moving up - find last word on a line < currentLine
				for i, wp := range readingState.WordPositions {
					if wp.LineIndex < currentLine {
						if targetWordIdx == -1 || wp.LineIndex > readingState.WordPositions[targetWordIdx].LineIndex {
							// Found a closer line
							targetWordIdx = i
							bestDistance = wp.WordInLine - currentWord.WordInLine
							if bestDistance < 0 {
								bestDistance = -bestDistance
							}
						} else if wp.LineIndex == readingState.WordPositions[targetWordIdx].LineIndex {
							// Same line as current target, check if closer word position
							distance := wp.WordInLine - currentWord.WordInLine
							if distance < 0 {
								distance = -distance
							}
							if distance < bestDistance {
								targetWordIdx = i
								bestDistance = distance
							}
						}
					}
				}
			}

			// If we found a word on another line, move to it
			if targetWordIdx != -1 {
				readingState.FocusedWordIndex = targetWordIdx

				// Ensure focused word is visible
				ensureWordVisible(readingState, viewportHeight)
			}
		},
		OnPageNavigation: func(delta int) {
			// Page navigation with h/l keys
			readingState.LastKeyWasG = false // Reset 'g' sequence

			if delta < 0 {
				// Previous page
				if readingState.CurrentPage > 0 {
					readingState.CurrentPage--
					updatePageWordPositions(state)
					loadPageIfNeeded(state, readingState.CurrentPage)
					// Save the new position
					saveReadingPosition(state)
				}
			} else {
				// Next page
				if readingState.CurrentPage < totalPages-1 {
					readingState.CurrentPage++
					updatePageWordPositions(state)
					loadPageIfNeeded(state, readingState.CurrentPage)
					// Pre-fetch next page
					if readingState.CurrentPage+1 < totalPages {
						loadPageIfNeeded(state, readingState.CurrentPage+1)
					}
					// Save the new position
					saveReadingPosition(state)
				}
			}
		},
		OnNextPage: func() {
			// Not used anymore, replaced by OnPageNavigation
		},
		OnPrevPage: func() {
			// Not used anymore, replaced by OnPageNavigation
		},
		OnKeyG: func() {
			// Handle 'g' key press for 'gg' sequence
			if readingState.LastKeyWasG {
				// Second 'g' press - jump to first word
				if len(readingState.WordPositions) > 0 {
					readingState.FocusedWordIndex = 0
					readingState.ScrollOffset = 0
				}
				readingState.LastKeyWasG = false
			} else {
				// First 'g' press - set flag
				readingState.LastKeyWasG = true
			}
		},
		OnJumpToFirst: func() {
			// Jump to first word (gg)
			if len(readingState.WordPositions) > 0 {
				readingState.FocusedWordIndex = 0
				readingState.ScrollOffset = 0
			}
			readingState.LastKeyWasG = false
		},
		OnJumpToLast: func() {
			// Jump to last word (G)
			if len(readingState.WordPositions) > 0 {
				readingState.FocusedWordIndex = len(readingState.WordPositions) - 1
				ensureWordVisible(readingState, viewportHeight)
			}
			readingState.LastKeyWasG = false
		},
		OnToggleDefinition: func() {
			log.Infof(context.Background(), "DEBUG OnToggleDefinition called, ShowDefinition=%v, FocusedWordIndex=%d, WordPositions=%d, DefinitionWord=%s", readingState.ShowDefinition, readingState.FocusedWordIndex, len(readingState.WordPositions), readingState.DefinitionWord)

			// Check if we have a valid focused word
			if len(readingState.WordPositions) == 0 || readingState.FocusedWordIndex >= len(readingState.WordPositions) {
				log.Infof(context.Background(), "DEBUG Cannot show definition: no words or invalid index")
				return
			}

			focusedWord := readingState.WordPositions[readingState.FocusedWordIndex]

			// If definition is showing and it's the same word, hide it (toggle off)
			if readingState.ShowDefinition && readingState.DefinitionWord == focusedWord.Word {
				readingState.ShowDefinition = false
				readingState.DefinitionWord = ""
				log.Infof(context.Background(), "DEBUG Definition hidden")
			} else {
				// Otherwise, show definition for the currently focused word (either new word or first time)
				readingState.ShowDefinition = true
				readingState.DefinitionWord = focusedWord.Word
				log.Infof(context.Background(), "DEBUG Definition shown for word: %s", focusedWord.Word)
			}
		},
	})
}

const PAGE_SIZE = 4096 // Each page is 4096 bytes

// parseWordPositions extracts word positions from content
func parseWordPositions(content string) []models.WordPosition {
	var positions []models.WordPosition
	lines := strings.Split(content, "\n")

	currentPos := 0
	for lineIdx, line := range lines {
		words := strings.Fields(line) // Split by whitespace
		if len(words) == 0 {
			currentPos += len(line) + 1 // +1 for newline
			continue
		}

		// Find each word in the line
		wordInLine := 0
		searchStart := 0
		for _, word := range words {
			// Find the word's position in the line
			wordStart := strings.Index(line[searchStart:], word)
			if wordStart == -1 {
				continue
			}
			wordStart += searchStart

			positions = append(positions, models.WordPosition{
				Word:       word,
				LineIndex:  lineIdx,
				WordInLine: wordInLine,
				StartPos:   currentPos + wordStart,
				EndPos:     currentPos + wordStart + len(word),
			})

			searchStart = wordStart + len(word)
			wordInLine++
		}

		currentPos += len(line) + 1 // +1 for newline
	}

	return positions
}

func loadPage(ctx context.Context, state *State, pageNum int) error {
	// Check if already in cache
	if _, exists := state.Reading.ContentCache[pageNum]; exists {
		return nil
	}

	readingState := &state.Reading
	readingState.Loading = true

	if readingState.LoadContent == nil {
		readingState.Error = "LoadContent is not set"
		readingState.Loading = false
		return nil
	}

	offset := pageNum * PAGE_SIZE
	content, totalBytes, lastOffset, err := readingState.LoadContent(ctx, readingState.MaterialID, offset, 10240) // Load 10240 bytes at once
	if err != nil {
		readingState.Error = err.Error()
		readingState.Loading = false
		return err
	}

	// Escape control characters in the content to prevent terminal issues
	// This must be done before caching and parsing word positions
	content = text.EscapeControlChars(content)

	readingState.TotalBytes = totalBytes
	readingState.Loading = false

	// If this is the first load (page 0) and there's a saved offset, navigate to it
	if pageNum == 0 && lastOffset > 0 {
		// Calculate the page number from the offset
		savedPage := int(lastOffset / int64(PAGE_SIZE))
		totalPages := (totalBytes + PAGE_SIZE - 1) / PAGE_SIZE
		if savedPage < totalPages {
			readingState.CurrentPage = savedPage
			// Load the saved page
			if _, exists := readingState.ContentCache[savedPage]; !exists {
				// Recursively load the saved page
				return loadPage(ctx, state, savedPage)
			}
		}
	}

	// Split the loaded content into pages and cache them
	for i := 0; i < len(content); i += PAGE_SIZE {
		end := i + PAGE_SIZE
		if end > len(content) {
			end = len(content)
		}
		pageContent := content[i:end]
		cachePageNum := pageNum + (i / PAGE_SIZE)
		readingState.ContentCache[cachePageNum] = pageContent
	}

	// If this is the current page, parse word positions
	if pageNum == readingState.CurrentPage {
		pageContent := readingState.ContentCache[pageNum]
		readingState.WordPositions = parseWordPositions(pageContent)
		readingState.FocusedWordIndex = 0
	}

	return nil
}

// ensureWordVisible adjusts scroll offset to ensure the focused word is visible in the viewport
func ensureWordVisible(readingState *ReadingState, viewportHeight int) {
	if len(readingState.WordPositions) == 0 {
		return
	}

	focusedWord := readingState.WordPositions[readingState.FocusedWordIndex]
	focusedLine := focusedWord.LineIndex

	// Check if focused line is above viewport
	if focusedLine < readingState.ScrollOffset {
		readingState.ScrollOffset = focusedLine
	}

	// Check if focused line is below viewport
	if focusedLine >= readingState.ScrollOffset+viewportHeight {
		readingState.ScrollOffset = focusedLine - viewportHeight + 1
	}

	// Ensure scroll offset is not negative
	if readingState.ScrollOffset < 0 {
		readingState.ScrollOffset = 0
	}
}

// updatePageWordPositions updates word positions when page changes
func updatePageWordPositions(state *State) {
	readingState := &state.Reading
	if pageContent, exists := readingState.ContentCache[readingState.CurrentPage]; exists {
		readingState.WordPositions = parseWordPositions(pageContent)
		readingState.FocusedWordIndex = 0
		readingState.ScrollOffset = 0 // Reset scroll to top when changing pages
	} else {
		readingState.WordPositions = nil
		readingState.FocusedWordIndex = 0
		readingState.ScrollOffset = 0
	}
}

// loadPageIfNeeded loads a page if it's not in cache
func loadPageIfNeeded(state *State, pageNum int) {
	if _, exists := state.Reading.ContentCache[pageNum]; !exists {
		state.Enqueue(func(ctx context.Context) error {
			return loadPage(ctx, state, pageNum)
		})
	}
}

// saveReadingPosition saves the current reading position to the backend
func saveReadingPosition(state *State) {
	readingState := &state.Reading
	if readingState.SavePosition == nil {
		return
	}

	// Calculate byte offset from current page
	offset := int64(readingState.CurrentPage * PAGE_SIZE)

	// Save position asynchronously
	state.Enqueue(func(ctx context.Context) error {
		return readingState.SavePosition(ctx, readingState.MaterialID, offset)
	})
}

// HelpPage renders the help page with scrolling support
func HelpPage(state *State, window *dom.Window) *dom.Node {
	route := state.Routes.Last()
	helpState := route.HelpPage

	// Calculate viewport height from window dimensions
	// Reserve space for title (1 line), exit message (1 line), status bar (1 line), and some padding
	reservedLines := 4
	viewportHeight := window.Height - reservedLines
	if viewportHeight < 5 {
		viewportHeight = 5 // Minimum viewport height
	}

	return dom.Div(dom.DivProps{
		Focusable: true,
		Focused:   true,
		OnKeyDown: func(event *dom.DOMEvent) {
			keyEvent := event.KeydownEvent
			if keyEvent == nil {
				return
			}

			log.Infof(context.Background(), "key: %v", keyEvent.KeyType)

			switch keyEvent.KeyType {
			case dom.KeyTypeUp:
				// Scroll up
				if helpState.ScrollOffset > 0 {
					helpState.ScrollOffset--
				}
				event.PreventDefault()
			case dom.KeyTypeDown:
				// Scroll down with bounds checking
				totalLines := help.GetTotalLines()
				maxScroll := totalLines - viewportHeight
				if maxScroll < 0 {
					maxScroll = 0
				}
				if helpState.ScrollOffset < maxScroll {
					helpState.ScrollOffset++
				}
				event.PreventDefault()
			}
		},
	}, help.Help(help.HelpProps{
		ScrollOffset:   helpState.ScrollOffset,
		ViewportHeight: viewportHeight,
	}))
}

func RenderRoute(state *State, route Route, window *dom.Window) *dom.Node {
	// Fixed frame overhead: Title (2) + Exit message (1) + Status bar (1) + Spacing (1) = 5 lines
	const FIXED_FRAME_HEIGHT = 5
	availableHeight := window.Height - FIXED_FRAME_HEIGHT
	if availableHeight < 5 {
		availableHeight = 5 // Minimum height
	}

	switch route.Type {
	case RouteType_Detail:
		return DetailPage(state, route.DetailPage.EntryID)
	case RouteType_Config:
		return ConfigPage(state)
	case RouteType_HappeningList:
		return HappeningListPage(state)
	case RouteType_HumanState:
		return HumanStatePage(state)
	case RouteType_Help:
		return HelpPage(state, window)
	case RouteType_Learning:
		return LearningPage(state, window.Width, availableHeight)
	case RouteType_Reading:
		return ReadingPage(state, route.ReadingPage.MaterialID, window.Width, availableHeight)
	default:
		return dom.Text(fmt.Sprintf("unknown route: %d", route.Type), styles.Style{
			Bold:  true,
			Color: colors.RED_ERROR,
		})
	}
}
