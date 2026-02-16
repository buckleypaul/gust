package west

import (
	"bufio"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Board represents a parsed board entry from `west boards`.
type Board struct {
	Name         string
	Architecture string
	Qualifiers   string
}

// BoardsLoadedMsg is sent when the board list has been parsed.
type BoardsLoadedMsg struct {
	Boards []Board
	Err    error
}

// ListBoards runs `west boards` and parses the output into Board structs.
func ListBoards() tea.Cmd {
	return func() tea.Msg {
		output, err := RunSimple("west", "boards")
		if err != nil {
			return BoardsLoadedMsg{Err: err}
		}
		boards := parseBoards(output)
		return BoardsLoadedMsg{Boards: boards}
	}
}

func parseBoards(output string) []Board {
	var boards []Board
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// west boards output format varies by Zephyr version.
		// Common format: board_name
		// Some versions: board_name  arch  qualifiers
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		b := Board{Name: fields[0]}
		if len(fields) > 1 {
			b.Architecture = fields[1]
		}
		if len(fields) > 2 {
			b.Qualifiers = strings.Join(fields[2:], " ")
		}
		boards = append(boards, b)
	}
	return boards
}
