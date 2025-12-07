package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
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
	padding                   = 2
	spinnerFPS                = 1
	minStartProgress          = 0.02
	maxBeforeFinishProgress   = 0.95
	maxWidth                  = 80
	maxPomodoroCycle          = 4
	defaultContextHint        = "s · start/pause | r · reset | b · next"
	defaultMinimalContextHint = "b · next"
)

type PomodoroState int

const (
	StateFocus PomodoroState = iota
	StateBreak
	StateLongBreak
)

var stateName = map[PomodoroState]string{
	StateFocus:     "StateFocus",
	StateBreak:     "StateBreak",
	StateLongBreak: "StateLongBreak",
}

var stateLabel = map[PomodoroState]string{
	StateFocus:     "Focus",
	StateBreak:     "Break",
	StateLongBreak: "Long Break",
}

func (state PomodoroState) String() string {
	return stateName[state]
}

func getStateLabel(state PomodoroState) string {
	return stateLabel[state]
}

type tickMsg time.Time

type model struct {
	// Setting
	keymap               keymap
	setting              setting
	// Visual
	progress             progress.Model
	spinner              spinner.Model
	windowWidth			 int
	windowHeight		 int
	// App State
	isStart              bool
	isPause              bool
	isFinish             bool
	isUseMinimalHint     bool
	currentTimeSeconds   float64
	currentPomodoroCycle int
	currentState         PomodoroState
}

type setting struct {
	focusTime      int
	breakTime      int
	longBreakTime  int
	useMinimalHint bool
}

type keymap struct {
	start      key.Binding
	reset      key.Binding
	hardReset  key.Binding
	next       key.Binding
	toggleHint key.Binding
	quit       key.Binding
}

var activeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render
var inactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
var layoutStyle = lipgloss.NewStyle().Padding(1, 1).Render
var progressOption = progress.WithSolidFill("#ffffff")

var alertScriptPath string = ""

func getPomodoroTime(state PomodoroState) int {
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

func getDefaultSpinner(state PomodoroState, fps int) spinner.Spinner {
	var result spinner.Spinner

	switch state {
	case StateFocus:
		result = spinner.Hamburger
	default:
		result = spinner.Line
	}

	result.FPS = (time.Second / time.Duration(fps))
	return result
}

func main() {

	alertScriptPath = os.Getenv("tPOMODORO_ALERT_SCRIPT")
	m := model{
		progress: progress.New(progressOption),
		spinner:  spinner.New(spinner.WithSpinner(getDefaultSpinner(StateFocus, spinnerFPS))),
		windowWidth: 80,
		windowHeight: 40,
		setting: setting{
			focusTime:     		getPomodoroTime(StateFocus),
			breakTime:			getPomodoroTime(StateBreak),
			longBreakTime:		getPomodoroTime(StateLongBreak),
			useMinimalHint:		false,
		},
		isUseMinimalHint:	  false,
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
			hardReset: key.NewBinding(
				key.WithKeys("ctrl+r"),
				key.WithHelp("ctrl+r", "hard reset"),
			),
			next: key.NewBinding(
				key.WithKeys("b"),
				key.WithHelp("b", "next"),
			),
			toggleHint: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "toggle hint"),
			),
			quit: key.NewBinding(
				key.WithKeys("ctrl+c", "q"),
				key.WithHelp("q", "quit"),
			),
		},
	}

	m.progress.Width = maxWidth
	m.progress.ShowPercentage = false
	m.spinner.Spinner.FPS = (time.Second / spinnerFPS)

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return tea.SetWindowTitle("tPomodoro")
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		// Hard Reset
		case key.Matches(msg, m.keymap.hardReset):
			m.isStart = false
			m.isPause = false
			m.isFinish = false
			m.currentState = StateFocus
			m.currentTimeSeconds = 0
			m.currentPomodoroCycle = 1
			m.spinner.Spinner = getDefaultSpinner(StateFocus, spinnerFPS)
			cmdProgress := m.progress.SetPercent(0)
			return m, cmdProgress

		// Stop and Reset Timer
		case key.Matches(msg, m.keymap.reset):
			m.isStart = false
			m.isPause = false
			m.isFinish = false
			m.currentTimeSeconds = 0
			cmdProgress := m.progress.SetPercent(0)
			return m, cmdProgress

		// Start/Pause Timer
		case key.Matches(msg, m.keymap.start):
			isStart := m.isStart

			if isStart {
				isPaused := !m.isPause
				m.isPause = isPaused

				if isPaused {
					return m, nil
				} else {
					cmd := m.spinner.Tick
					return m, cmd
				}

			} else {
				m.isStart = true
				return m, tea.Batch(tickCmd(), m.spinner.Tick)
			}

		// Change Pomodoro State
		case key.Matches(msg, m.keymap.next):
			nextState := StateFocus
			nextPomodoroCycle := m.currentPomodoroCycle

			switch m.currentState {
			case StateFocus:
				shouldChangeToLongBreak := ((m.currentPomodoroCycle + 1) > maxPomodoroCycle)

				if shouldChangeToLongBreak {
					nextState = StateLongBreak
				} else {
					nextState = StateBreak
				}

			case StateBreak:
				nextState = StateFocus
				nextPomodoroCycle = (m.currentPomodoroCycle + 1)

			case StateLongBreak:
				nextState = StateFocus
				nextPomodoroCycle = 1
			}

			m.isFinish = false
			m.currentTimeSeconds = 0
			m.currentState = nextState
			m.currentPomodoroCycle = nextPomodoroCycle
			m.spinner.Spinner = getDefaultSpinner(nextState, spinnerFPS)

			cmd := m.progress.SetPercent(0)

			if m.isPause {
				m.isPause = false
				return m, tea.Batch(cmd, m.spinner.Tick)
			}

			return m, cmd

		// Toggle Hint Style
		case key.Matches(msg, m.keymap.toggleHint):
			isUseMinimalHint := !m.isUseMinimalHint
			m.isUseMinimalHint = isUseMinimalHint
			return m, nil

		// Quit
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit

		default:
			return m, nil
		}

	case tea.WindowSizeMsg:

		expectWidth := msg.Width - padding*2 - 4
		m.progress.Width = min(expectWidth, maxWidth)
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height

		return m, nil

	case tickMsg:

		if !m.isStart {
			return m, nil
		}

		if m.isPause {
			return m, tickCmd()
		}

		targetTimeSeconds := float64(getPomodoroTime(m.currentState) * 60.0)
		currentTimeSeconds := (m.currentTimeSeconds + 1)

		if currentTimeSeconds > targetTimeSeconds {
			currentTimeSeconds = targetTimeSeconds
		}

		m.currentTimeSeconds = currentTimeSeconds
		shouldFinish := !m.isFinish && (targetTimeSeconds == currentTimeSeconds)

		if shouldFinish {
			m.isFinish = true
			executeShell(alertScriptPath, m.currentState.String())
		}

		timeProgress := (currentTimeSeconds / targetTimeSeconds)
		visualTimeProgress := timeProgress

		shouldEarlyFillProgress := !m.isFinish && (visualTimeProgress < minStartProgress)
		shouldPreventFullFillProgress := !m.isFinish && (visualTimeProgress > maxBeforeFinishProgress)

		if shouldEarlyFillProgress {
			visualTimeProgress = minStartProgress
		} else if shouldPreventFullFillProgress {
			visualTimeProgress = maxBeforeFinishProgress
		}

		cmd := m.progress.SetPercent(visualTimeProgress)
		return m, tea.Batch(tickCmd(), cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		shouldUpdateSpinner := m.isStart && !m.isPause

		if shouldUpdateSpinner {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

		return m, nil
	}
}

func (m model) View() string {
	var title string
	var subTitle string
	var contextHint string

	shouldShowPomodoroCycle := (m.currentState != StateLongBreak)
	statelabel := getStateLabel(m.currentState)

	if shouldShowPomodoroCycle {
		title = fmt.Sprintf("%s (%d/%d)", statelabel, m.currentPomodoroCycle, maxPomodoroCycle)
	} else {
		title = statelabel
	}

	if m.isPause {
		subTitle = "paused"
	} else if m.progress.Percent() == 1.0 {
		subTitle = "done"
	} else {
		targetTimeMinutes := getPomodoroTime(m.currentState)
		currentTimeMinutes := int(math.Trunc(m.currentTimeSeconds / 60.0))
		elapsed := targetTimeMinutes - currentTimeMinutes
		subTitle = fmt.Sprintf("%2dm", elapsed)
	}

	if m.isUseMinimalHint {
		if m.progress.Percent() == 1.0 {
			contextHint = defaultMinimalContextHint
		} else {
			contextHint = ""
		}
	} else {
		contextHint = defaultContextHint
	}

	titlePadding := utf8.RuneCountInString(title)
	subTitlePadding := utf8.RuneCountInString(subTitle) + 2

	totalTopPadding := (m.progress.Width - (titlePadding + subTitlePadding))
	totalBottomPadding := (m.progress.Width - utf8.RuneCountInString(contextHint))

	if totalTopPadding < 0 {
		totalTopPadding = 0
	}

	if totalBottomPadding < 0 {
		totalBottomPadding = 0
	}

	extraPadTop := strings.Repeat(" ", totalTopPadding)
	extraPadBottom := strings.Repeat(" ", totalBottomPadding)

	if m.isStart {
		subTitle = activeStyle(subTitle)
	} else {
		subTitle = inactiveStyle(subTitle)
	}

	views := m.spinner.View() + " " + activeStyle(title) + extraPadTop + subTitle + "\n" +
		m.progress.View() + "\n" +
		extraPadBottom + inactiveStyle(contextHint)

	return lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, lipgloss.Center, layoutStyle(views))

}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func executeShell(scriptPath string, message string) {
	if scriptPath == "" {
		return
	}

	cmd := exec.Command(scriptPath, message)
	err := cmd.Run()

	if err != nil {
		log.Fatalf("\nCommand failed: %v", err)
	}
}

