package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

const notesFolder = "/Users/torbratsberg/main/notes/"

type model struct {
	fileContents string
	readerView   viewport.Model
	notes        []fs.DirEntry
	listView     viewport.Model
	pointer      int
	ready        bool
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
		if note.IsDir() {
			continue
		}

		if i == m.pointer {
			listStr += fmt.Sprintf("> %s\n", note.Name())
		} else {
			listStr += fmt.Sprintf("  %s\n", note.Name())
		}
	}

	return listStr
}

func (m model) RenderFile() string {
	if m.cachedFiles[m.notes[m.pointer].Name()] != "" {
		return m.cachedFiles[m.notes[m.pointer].Name()]
	}

	content, err := ioutil.ReadFile(notesFolder + m.notes[m.pointer].Name())
	check(err)
	out, err := glamour.Render(string(content), "dark")
	check(err)

	m.cachedFiles[m.notes[m.pointer].Name()] = out

	return out
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
		}

		if k == "k" || k == "j" {
			m.listView.SetContent("\n===\n\n" + m.RenderList())
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

			// go func() {
			// 	err := cmd.Run()
			// 	check(err)
			// }()

			return m, tea.Quit
		}

		return m, nil

	case tea.WindowSizeMsg:
		if !m.ready {
			m.readerView = viewport.New(msg.Width/2, msg.Height/2)
			m.readerView.SetContent(m.fileContents)

			m.listView = viewport.New(msg.Width/2, msg.Height/2)
			m.listView.SetContent("\n===\n\n" + m.RenderList())

			m.ready = true
		} else {
			m.listView.Width = msg.Width / 2
			m.listView.Height = msg.Height / 2
			m.readerView.Width = msg.Width / 2
			m.readerView.Height = msg.Height / 2
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

	// Load some text for our viewport
	var content string
	if len(notes) > 0 {
		contentBytes, err := ioutil.ReadFile(notesFolder + notes[0].Name())
		check(err)
		parsed, err := glamour.Render(string(contentBytes), "dark")
		check(err)
		content = parsed
	} else {
		parsed, err := glamour.Render(
			fmt.Sprintf(
				"No notes found in %s.\nPress `n` to create a new note.\n",
				notesFolder,
			),
			"dark",
		)

		check(err)
		content = parsed
	}

	p := tea.NewProgram(
		model{
			fileContents: string(content),
			notes:        notes,
			cachedFiles:  make(map[string]string, len(notes)),
		},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	err = p.Start()
	check(err)
}
