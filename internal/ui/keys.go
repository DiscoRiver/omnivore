package ui

import "charm.land/bubbles/v2/key"

// keyMap holds every key binding, so that the help footer is generated from the same
// definitions the UI responds to.
type keyMap struct {
	Up, Down, PageUp, PageDown, Top, Bottom key.Binding
	Open, Diff, Next, Prev, Follow, Back    key.Binding
	Search, Filter, FilterBack, Copy, Log   key.Binding
	Help, Quit                              key.Binding
	// In overlays.
	Confirm, Cancel key.Binding
}

func newKeyMap() keyMap {
	b := func(keys []string, k, desc string) key.Binding {
		return key.NewBinding(key.WithKeys(keys...), key.WithHelp(k, desc))
	}
	return keyMap{
		Up:         b([]string{"up", "k"}, "↑/k", "up"),
		Down:       b([]string{"down", "j"}, "↓/j", "down"),
		PageUp:     b([]string{"pgup"}, "pgup", "page up"),
		PageDown:   b([]string{"pgdown", "space"}, "pgdn", "page down"),
		Top:        b([]string{"home", "g"}, "g", "top"),
		Bottom:     b([]string{"end", "G"}, "G", "bottom"),
		Open:       b([]string{"enter"}, "enter", "open"),
		Diff:       b([]string{"d"}, "d", "diff vs majority"),
		Next:       b([]string{"right", "n"}, "→/n", "next host"),
		Prev:       b([]string{"left", "p"}, "←/p", "prev host"),
		Follow:     b([]string{"f"}, "f", "follow output"),
		Back:       b([]string{"esc"}, "esc", "back"),
		Search:     b([]string{"/"}, "/", "find host"),
		Filter:     b([]string{"tab"}, "tab", "filter"),
		FilterBack: b([]string{"shift+tab"}, "shift+tab", "filter back"),
		Copy:       b([]string{"y"}, "y", "copy hosts"),
		Log:        b([]string{"l"}, "l", "log"),
		Help:       b([]string{"?"}, "?", "more keys"),
		Quit:       b([]string{"q", "ctrl+c"}, "q", "quit"),
		Confirm:    b([]string{"y", "enter", "ctrl+c"}, "y", "quit"),
		Cancel:     b([]string{"n", "esc"}, "n", "stay"),
	}
}

// helpKeys is the subset of bindings that applies on one screen.
type helpKeys struct {
	short []key.Binding
	full  [][]key.Binding
}

func (h helpKeys) ShortHelp() []key.Binding  { return h.short }
func (h helpKeys) FullHelp() [][]key.Binding { return h.full }

// help returns the bindings to show for the current screen and overlay.
func (k keyMap) help(s screen, o overlay) helpKeys {
	scroll := []key.Binding{k.Up, k.Down, k.PageUp, k.PageDown, k.Top, k.Bottom}
	global := []key.Binding{k.Search, k.Filter, k.FilterBack, k.Log, k.Help, k.Quit}
	switch o {
	case searchOverlay:
		return helpKeys{short: []key.Binding{k.Up, k.Down, k.Open, k.Back}}
	case logOverlay:
		return helpKeys{short: []key.Binding{k.Up, k.Down, k.PageUp, k.PageDown, k.Log, k.Back}}
	case quitOverlay:
		return helpKeys{short: []key.Binding{k.Confirm, k.Cancel}}
	}
	switch s {
	case hostScreen:
		return helpKeys{
			short: []key.Binding{k.Prev, k.Next, k.PageDown, k.Follow, k.Back, k.Search, k.Help, k.Quit},
			full:  [][]key.Binding{{k.Prev, k.Next, k.Follow, k.Copy, k.Back}, scroll, global},
		}
	case diffScreen:
		return helpKeys{
			short: []key.Binding{k.Up, k.Down, k.Open, k.Back, k.Help, k.Quit},
			full:  [][]key.Binding{{k.Open, k.Copy, k.Back}, scroll, global},
		}
	default:
		return helpKeys{
			short: []key.Binding{k.Up, k.Down, k.Open, k.Diff, k.Search, k.Filter, k.Log, k.Help, k.Quit},
			full:  [][]key.Binding{{k.Up, k.Down, k.Open, k.Diff, k.Next, k.Copy}, {k.PageUp, k.PageDown, k.Top, k.Bottom}, global},
		}
	}
}
