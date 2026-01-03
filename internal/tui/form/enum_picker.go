package form

import (
	"strings"
)

type enumItem struct {
	name  string
	value string
}

// enumPicker displays all enum values inline as [ val1, val2, val3 ]
// with the selected value highlighted.
type enumPicker struct {
	items    []enumItem
	selected int
}

func newEnumPicker(items []enumItem) enumPicker {
	return enumPicker{
		items:    items,
		selected: 0,
	}
}

func (p *enumPicker) Next() {
	if len(p.items) == 0 {
		return
	}
	p.selected = (p.selected + 1) % len(p.items)
}

func (p *enumPicker) Prev() {
	if len(p.items) == 0 {
		return
	}
	p.selected--
	if p.selected < 0 {
		p.selected = len(p.items) - 1
	}
}

func (p *enumPicker) SelectedItem() *enumItem {
	if len(p.items) == 0 || p.selected < 0 || p.selected >= len(p.items) {
		return nil
	}
	return &p.items[p.selected]
}

func (p *enumPicker) View() string {
	if len(p.items) == 0 {
		return bracketStyle.Render("[ ]")
	}

	var b strings.Builder
	b.WriteString(bracketStyle.Render("[ "))

	for i, item := range p.items {
		if i == p.selected {
			b.WriteString(selectedStyle.Render(item.name))
		} else {
			b.WriteString(unselectedStyle.Render(item.name))
		}

		if i < len(p.items)-1 {
			b.WriteString(unselectedStyle.Render(", "))
		}
	}

	b.WriteString(bracketStyle.Render(" ]"))
	return b.String()
}
