package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"golang.org/x/crypto/ssh/terminal"
)

var notesFolder = os.Getenv("NOTES_DIR") + "/"

type model struct {
	note       string
	readerView viewport.Model
	notes      []fs.DirEntry
	listView   viewport.Model
	pointer    int
	ready      bool
	glammy     glamour.TermRenderer
	cache      map[string]string
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) OpenFile(fileName string) {
	// Open the file in $EDITOR
	cmd := exec.Command(
		os.Getenv("EDITOR"),
		notesFolder+fileName)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	check(err)
}

func (m model) RenderList() string {
	listStr := ""

	// Add pointer to the correct note
	for i, note := range m.notes {
		if i == m.pointer {
			listStr += fmt.Sprintf("> %s\n", note.Name())
		} else {
			listStr += fmt.Sprintf("  %s\n", note.Name())
		}
	}

	return listStr
}

func (m model) RenderFile() string {
	// Check if the file is cached
	if m.cache[m.notes[m.pointer].Name()] != "" {
		return m.cache[m.notes[m.pointer].Name()]
	}

	// Read and render the file
	content, err := ioutil.ReadFile(notesFolder + m.notes[m.pointer].Name())
	check(err)
	out, err := m.glammy.Render(string(content))
	check(err)

	// Cache the file
	m.cache[m.notes[m.pointer].Name()] = out

	return out
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()

		if k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}

		switch k {
		case "k":
			if m.pointer > 0 {
				m.pointer--
			}
		case "j":
			if len(m.notes)-1 > m.pointer {
				m.pointer++
			}
		case "d":
			m.readerView.HalfViewDown()
		case "u":
			m.readerView.HalfViewUp()
		case "n":
			m.OpenFile("new_note.md")
		case "enter":
			m.OpenFile(m.notes[m.pointer].Name())
			return m, tea.Quit
		}

		if k == "k" || k == "j" {
			m.listView.SetContent(fmt.Sprintf(
				"\n\n  %s  \n\n%s",
				strings.Repeat("=", m.readerView.Width-4),
				m.RenderList()))

			m.readerView.SetContent(m.RenderFile())
		}

		return m, nil

	case tea.WindowSizeMsg:
		if !m.ready {
			m.readerView = viewport.New(msg.Width, msg.Height/4*3)
			m.readerView.SetContent(m.note)

			m.listView = viewport.New(msg.Width, msg.Height/4)
			m.listView.SetContent(fmt.Sprintf(
				"\n\n  %s  \n\n%s",
				strings.Repeat("=", m.readerView.Width-4),
				m.RenderList()))

			m.ready = true
		} else {
			m.listView.Width = msg.Width
			m.listView.Height = msg.Height / 4
			m.readerView.Width = msg.Width
			m.readerView.Height = msg.Height / 4 * 3
		}
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s%s", m.readerView.View(), m.listView.View())
}

func main() {
	// Read our notes folder
	notes, err := os.ReadDir(notesFolder)
	check(err)

	// Filter out the dirs
	n := 0
	for _, note := range notes {
		if !note.IsDir() {
			notes[n] = note
			n++
		}
	}
	notes = notes[:n]

	termWidth, _, err := terminal.GetSize(0)
	check(err)
	glammy, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(termWidth-3))

	// Load the first note if there is one
	var content string
	if len(notes) > 0 {
		contentBytes, err := ioutil.ReadFile(notesFolder + notes[0].Name())
		parsed, err := glammy.Render(string(contentBytes))
		check(err)

		content = parsed
	} else {
		parsed, err := glammy.Render(fmt.Sprintf(
			"# No notes found in %s.\n\nPress `n` to create a new note.\n",
			notesFolder))

		check(err)
		content = parsed
	}

	// Initiate the program
	p := tea.NewProgram(
		model{
			note:   string(content),
			notes:  notes,
			cache:  make(map[string]string, len(notes)),
			glammy: *glammy,
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())

	err = p.Start()
	check(err)
}
