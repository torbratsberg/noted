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

const notesFolder = "/Users/torbratsberg/main/notes/"

type model struct {
	fileContents string
	readerView   viewport.Model
	notes        []fs.DirEntry
	listView     viewport.Model
	pointer      int
	ready        bool
	glammy       glamour.TermRenderer
	cachedFiles  map[string]string
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) RenderList() string {
	listStr := ""

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
	if m.cachedFiles[m.notes[m.pointer].Name()] != "" {
		return m.cachedFiles[m.notes[m.pointer].Name()]
	}

	// Read and render the file
	content, err := ioutil.ReadFile(notesFolder + m.notes[m.pointer].Name())
	check(err)
	out, err := m.glammy.Render(string(content))
	check(err)

	// Cache the file
	m.cachedFiles[m.notes[m.pointer].Name()] = out

	return out
}

func (m model) NewFile() {
	// Open the file in $EDITOR
	cmd := exec.Command(
		os.Getenv("EDITOR"),
		notesFolder+"my_new_note.md",
	)
	m.listView.SetContent(cmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	check(err)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()

		if k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		} else if k == "j" {
			if len(m.notes)-1 > m.pointer {
				m.pointer++
			}
		} else if k == "k" {
			if m.pointer > 0 {
				m.pointer--
			}
		} else if k == "d" {
			m.readerView.HalfViewDown()
		} else if k == "u" {
			m.readerView.HalfViewUp()
		} else if k == "n" {
			m.NewFile()

			// } else if k == "d" {
			// 	err := os.Remove(notesFolder + m.notes[m.pointer].Name())
			// 	check(err)
		}

		if k == "k" || k == "j" {
			m.listView.SetContent(fmt.Sprintf(
				"\n\n%s\n\n%s",
				strings.Repeat("=", m.readerView.Width),
				m.RenderList()))
			m.readerView.SetContent(m.RenderFile())
		}

		if k == "enter" {
			cmd := exec.Command(
				os.Getenv("EDITOR"),
				notesFolder+m.notes[m.pointer].Name(),
			)
			m.listView.SetContent(cmd.String())
			cmd.Stdout = os.Stdout
			// cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin

			err := cmd.Run()
			if err != nil {
				return m, tea.Quit
			}

			return m, tea.Quit
		}

		return m, nil

	case tea.WindowSizeMsg:
		if !m.ready {
			m.readerView = viewport.New(msg.Width, msg.Height/4*3)
			m.readerView.SetContent(m.fileContents)

			m.listView = viewport.New(msg.Width, msg.Height/4)
			m.listView.SetContent(fmt.Sprintf(
				"\n\n%s\n\n%s",
				strings.Repeat("=", m.readerView.Width),
				m.RenderList()))

			m.ready = true
		} else {
			m.listView.Width = msg.Width
			m.listView.Height = msg.Height / 4
			m.readerView.Width = msg.Width
			m.readerView.Height = msg.Height / 4 * 3
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.readerView, cmd = m.readerView.Update(msg)
	cmds = append(cmds, cmd)
	m.listView, cmd = m.listView.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s%s", m.readerView.View(), m.listView.View())
}

func main() {

	notes, err := os.ReadDir("/Users/torbratsberg/main/notes/")
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
		glamour.WithWordWrap(termWidth-3),
	)

	// Load some text for our viewport
	var content string
	if len(notes) > 0 {
		contentBytes, err := ioutil.ReadFile(notesFolder + notes[0].Name())
		check(err)
		parsed, err := glammy.Render(string(contentBytes))
		check(err)
		content = parsed
	} else {
		parsed, err := glammy.Render(fmt.Sprintf(
			"No notes found in %s.\nPress `n` to create a new note.\n",
			notesFolder,
		))

		check(err)
		content = parsed
	}

	p := tea.NewProgram(
		model{
			fileContents: string(content),
			notes:        notes,
			cachedFiles:  make(map[string]string, len(notes)),
			glammy:       *glammy,
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	err = p.Start()
	check(err)
}
