// TODO : better key binding (s - start/stop, r - reset, b - next)
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	padding          = 2
	spinnerFPS       = 2
	minStartProgress = 0.012
	maxWidth         = 80
	maxPomodoroCycle = 4
)

type PomodoroState int

const (
	StateFocus PomodoroState = iota
	StateBreak
	StateLongBreak
)

var stateLabel = map[PomodoroState]string{
	StateFocus:     "Focus",
	StateBreak:     "Break",
	StateLongBreak: "Long Break",
}

func (state PomodoroState) String() string {
	return stateLabel[state]
}

type tickMsg time.Time

type model struct {
	progress             progress.Model
	spinner              spinner.Model
	keymap               keymap
	setting              setting
	isStart              bool
	isPause              bool
	currentState         PomodoroState
	currentTimeSeconds   float64
	currentPomodoroCycle int
}

type setting struct {
	focusTime     int
	breakTime     int
	longBreakTime int
}

type keymap struct {
	start key.Binding
	reset key.Binding
	quit  key.Binding
}

var titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render
var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
var progressOption = progress.WithSolidFill("#ffffff")

func getDefaultTime(state PomodoroState) int {
	switch state {
	case StateFocus:
		return 25
	case StateBreak:
		return 5
	case StateLongBreak:
		return 30
	default:
		return -1
	}
}

func getDefaultSpinner(state PomodoroState) spinner.Spinner {
	switch state {
	case StateFocus:
		return spinner.Hamburger
	default:
		return spinner.Line
	}
}

func main() {
	m := model{
		progress: progress.New(progressOption),
		spinner:  spinner.New(spinner.WithSpinner(getDefaultSpinner(StateFocus))),
		setting: setting{
			focusTime:     getDefaultTime(StateFocus),
			breakTime:     getDefaultTime(StateBreak),
			longBreakTime: getDefaultTime(StateLongBreak),
		},
		currentState:         StateFocus,
		currentTimeSeconds:   0,
		currentPomodoroCycle: 1,
		keymap: keymap{
			start: key.NewBinding(
				key.WithKeys("s", " "),
				key.WithHelp("s", "start/pause"),
			),
			reset: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "reset"),
			),
			quit: key.NewBinding(
				key.WithKeys("ctrl+c", "q"),
				key.WithHelp("q", "quit"),
			),
		},
	}

	m.progress.Width = maxWidth
	m.progress.ShowPercentage = false
	m.spinner.Spinner.FPS = time.Second / spinnerFPS

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.reset):
			m.isStart = false
			m.isPause = false
			cmd := m.progress.SetPercent(0)
			return m, cmd
		case key.Matches(msg, m.keymap.start):
			isStart := m.isStart
			if isStart {
				isPaused := !m.isPause
				return onPaused(m, isPaused)
			} else {
				return onStarted(m, true)
			}
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		default:
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case tickMsg:

		if !m.isStart {
			return m, nil
		}

		// Note that you can also use progress.Model.SetPercent to set the
		if m.isPause {
			return m, tickCmd()
		}

		// percentage value explicitly, too.
		cmd := m.progress.IncrPercent(0.2)
		return m, tea.Batch(tickCmd(), cmd)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		if m.isStart && !m.isPause {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

func onStarted(m model, isStarted bool) (tea.Model, tea.Cmd) {
	m.isStart = isStarted
	return m, tea.Batch(tickCmd(), m.spinner.Tick)
}

func onPaused(m model, isPaused bool) (tea.Model, tea.Cmd) {
	m.isPause = isPaused

	if !isPaused {
		cmd := m.spinner.Tick
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	title := "Focus (1/4)"
	//title := "Break (1/4)"
	//title := "Long Break"
	var subTitle string
	var contextHint string

	if m.isPause {
		subTitle = "paused"
	} else if m.progress.Percent() == 1.0 {
		subTitle = "done"
	} else {
		subTitle = "25m"
		//subTitle = " 5m"
		//subTitle = "30m"
	}

	// TODO : context aware 's - start/pause/resume', show one word at the time
	//if m.progress.Percent() == 1.0 {
	contextHint = "s · start/pause | r · reset | b · next"
	//}

	titlePadding := utf8.RuneCountInString(title)
	subTitlePadding := utf8.RuneCountInString(subTitle) + 2

	totalBottomPadding := m.progress.Width - utf8.RuneCountInString(contextHint)

	if totalBottomPadding < 0 {
		totalBottomPadding = 0
	}

	pad := strings.Repeat(" ", padding)
	extraPadTop := strings.Repeat(" ", m.progress.Width-(titlePadding+subTitlePadding))
	extraPadBottom := strings.Repeat(" ", totalBottomPadding)

	if m.isStart {
		subTitle = titleStyle(subTitle)
	} else {
		subTitle = helpStyle(subTitle)
	}

	return "\n" +
		pad + m.spinner.View() + " " + titleStyle(title) + extraPadTop + titleStyle(subTitle) + "\n" +
		pad + m.progress.View() + "\n" +
		pad + extraPadBottom + helpStyle(contextHint)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
