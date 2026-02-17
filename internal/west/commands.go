package west

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Status runs `west status` and returns the output.
func Status() tea.Cmd {
	return RunStreaming("west", "status")
}

// List runs `west list` and returns the output.
func List() tea.Cmd {
	return RunStreaming("west", "list")
}

// Diff runs `west diff` and returns the output.
func Diff() tea.Cmd {
	return RunStreaming("west", "diff")
}

// Forall runs `west forall -c <cmd>` and returns the output.
func Forall(cmd string) tea.Cmd {
	return RunStreaming("west", "forall", "-c", cmd)
}

// Update runs `west update` and streams output.
func Update() tea.Cmd {
	return RunStreaming("west", "update")
}

// Init runs `west init -l .` to initialize the workspace using the local manifest.
func Init() tea.Cmd {
	return RunStreaming("west", "init", "-l", ".")
}

// ZephyrExport runs `west zephyr-export` to export CMake packages.
func ZephyrExport() tea.Cmd {
	return RunStreaming("west", "zephyr-export")
}

// PackagesPipInstall runs `west packages pip --install` to install Python dependencies.
func PackagesPipInstall() tea.Cmd {
	return RunStreaming("west", "packages", "pip", "--install")
}

// SdkInstall runs `west sdk install` to download and install the Zephyr SDK.
// It first checks if wget is available, as the SDK setup script requires it.
// Installs only the ARM toolchain by default (covers most embedded targets like nRF, STM32, etc.)
func SdkInstall() tea.Cmd {
	return func() tea.Msg {
		// Pre-flight check: verify wget is installed
		if _, err := exec.LookPath("wget"); err != nil {
			return CommandResultMsg{
				Output: `ERROR: wget is required for Zephyr SDK installation but is not installed.

Please install wget first:
  brew install wget

Then re-run the setup wizard.
`,
				ExitCode: 1,
				Duration: 0,
			}
		}

		// Install SDK with ARM toolchain only (most common embedded targets)
		// User can install additional toolchains later with: west sdk install -t <toolchain>
		return RunStreaming("west", "sdk", "install", "-t", "arm-zephyr-eabi")()
	}
}

// requiredBrewPackages lists all Homebrew packages required for Zephyr development.
var requiredBrewPackages = []string{
	"cmake",
	"ninja",
	"gperf",
	"python3",
	"python-tk",
	"ccache",
	"qemu",
	"dtc",
	"libmagic",
	"wget",
	"openocd",
}

// InstallBrewDeps checks for required Homebrew packages and installs any that are missing.
func InstallBrewDeps() tea.Cmd {
	return func() tea.Msg {
		var output bytes.Buffer

		// Check if Homebrew is installed
		if _, err := exec.LookPath("brew"); err != nil {
			return CommandResultMsg{
				Output: `ERROR: Homebrew is not installed.

Please install Homebrew first:
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

Then re-run the setup wizard.
`,
				ExitCode: 1,
				Duration: 0,
			}
		}

		output.WriteString("Checking for required system dependencies...\n\n")

		// Check which packages are missing
		var missing []string
		for _, pkg := range requiredBrewPackages {
			cmd := exec.Command("brew", "list", pkg)
			if err := cmd.Run(); err != nil {
				missing = append(missing, pkg)
				output.WriteString(fmt.Sprintf("  ✗ %s (not installed)\n", pkg))
			} else {
				output.WriteString(fmt.Sprintf("  ✓ %s\n", pkg))
			}
		}

		if len(missing) == 0 {
			output.WriteString("\nAll required dependencies are installed!\n")
			return CommandResultMsg{
				Output:   output.String(),
				ExitCode: 0,
				Duration: 0,
			}
		}

		// Install missing packages
		output.WriteString(fmt.Sprintf("\nInstalling %d missing package(s): %s\n\n",
			len(missing), strings.Join(missing, ", ")))

		cmd := exec.Command("brew", append([]string{"install"}, missing...)...)
		applyEnv(cmd)
		cmd.Stdout = &output
		cmd.Stderr = &output

		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return CommandResultMsg{
					Output:   output.String(),
					ExitCode: exitErr.ExitCode(),
					Duration: 0,
				}
			}
			return CommandResultMsg{
				Output:   output.String() + fmt.Sprintf("\nError: %v\n", err),
				ExitCode: 1,
				Duration: 0,
			}
		}

		output.WriteString("\nAll dependencies installed successfully!\n")
		return CommandResultMsg{
			Output:   output.String(),
			ExitCode: 0,
			Duration: 0,
		}
	}
}
