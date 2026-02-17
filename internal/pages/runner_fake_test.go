package pages

import tea "github.com/charmbracelet/bubbletea"

type runCall struct {
	name string
	args []string
}

type fakeRunner struct {
	nextMsg tea.Msg

	runCalls             []runCall
	statusCalls          int
	listCalls            int
	diffCalls            int
	updateCalls          int
	initCalls            int
	zephyrExportCalls    int
	packagesPipCalls     int
	sdkInstallCalls      int
	installBrewDepsCalls int
}

func (f *fakeRunner) cmd() tea.Cmd {
	return func() tea.Msg {
		return f.nextMsg
	}
}

func (f *fakeRunner) Run(name string, args ...string) tea.Cmd {
	copied := append([]string(nil), args...)
	f.runCalls = append(f.runCalls, runCall{name: name, args: copied})
	return f.cmd()
}

func (f *fakeRunner) Status() tea.Cmd {
	f.statusCalls++
	return f.cmd()
}

func (f *fakeRunner) List() tea.Cmd {
	f.listCalls++
	return f.cmd()
}

func (f *fakeRunner) Diff() tea.Cmd {
	f.diffCalls++
	return f.cmd()
}

func (f *fakeRunner) Update() tea.Cmd {
	f.updateCalls++
	return f.cmd()
}

func (f *fakeRunner) Init() tea.Cmd {
	f.initCalls++
	return f.cmd()
}

func (f *fakeRunner) ZephyrExport() tea.Cmd {
	f.zephyrExportCalls++
	return f.cmd()
}

func (f *fakeRunner) PackagesPipInstall() tea.Cmd {
	f.packagesPipCalls++
	return f.cmd()
}

func (f *fakeRunner) SdkInstall() tea.Cmd {
	f.sdkInstallCalls++
	return f.cmd()
}

func (f *fakeRunner) InstallBrewDeps() tea.Cmd {
	f.installBrewDepsCalls++
	return f.cmd()
}
