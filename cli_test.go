package cli

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/posener/complete"
)

func TestCLIIsHelp(t *testing.T) {
	testCases := []struct {
		args   []string
		isHelp bool
	}{
		{[]string{"-h"}, true},
		{[]string{"-help"}, true},
		{[]string{"--help"}, true},
		{[]string{"-h", "foo"}, true},
		{[]string{"foo", "bar"}, false},
		{[]string{"-v", "bar"}, false},
		{[]string{"foo", "-h"}, true},
		{[]string{"foo", "-help"}, true},
		{[]string{"foo", "--help"}, true},
		{[]string{"foo", "bar", "-h"}, true},
		{[]string{"foo", "bar", "-help"}, true},
		{[]string{"foo", "bar", "--help"}, true},
		{[]string{"foo", "bar", "--", "zip", "-h"}, false},
		{[]string{"foo", "bar", "--", "zip", "-help"}, false},
		{[]string{"foo", "bar", "--", "zip", "--help"}, false},
	}

	for _, testCase := range testCases {
		cli := &CLI{Args: testCase.args}
		result := cli.IsHelp()

		if result != testCase.isHelp {
			t.Errorf("Expected '%#v'. Args: %#v", testCase.isHelp, testCase.args)
		}
	}
}

func TestCLIIsVersion(t *testing.T) {
	testCases := []struct {
		args      []string
		isVersion bool
	}{
		{[]string{"--", "-v"}, false},
		{[]string{"--", "-version"}, false},
		{[]string{"--", "--version"}, false},
		{[]string{"-v"}, true},
		{[]string{"-version"}, true},
		{[]string{"--version"}, true},
		{[]string{"-v", "foo"}, true},
		{[]string{"foo", "bar"}, false},
		{[]string{"-h", "bar"}, false},
		{[]string{"foo", "-v"}, false},
		{[]string{"foo", "-version"}, false},
		{[]string{"foo", "--version"}, false},
		{[]string{"foo", "--", "zip", "-v"}, false},
		{[]string{"foo", "--", "zip", "-version"}, false},
		{[]string{"foo", "--", "zip", "--version"}, false},
	}

	for _, testCase := range testCases {
		cli := &CLI{Args: testCase.args}
		result := cli.IsVersion()

		if result != testCase.isVersion {
			t.Errorf("Expected '%#v'. Args: %#v", testCase.isVersion, testCase.args)
		}
	}
}

func TestCLIRun(t *testing.T) {
	command := new(MockCommand)
	cli := &CLI{
		Args: []string{"foo", "-bar", "-baz"},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return command, nil
			},
		},
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != command.RunResult {
		t.Fatalf("bad: %d", exitCode)
	}

	if !command.RunCalled {
		t.Fatalf("run should be called")
	}

	if !reflect.DeepEqual(command.RunArgs, []string{"-bar", "-baz"}) {
		t.Fatalf("bad args: %#v", command.RunArgs)
	}
}

func TestCLIRun_blank(t *testing.T) {
	command := new(MockCommand)
	cli := &CLI{
		Args: []string{"", "foo", "-bar", "-baz"},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return command, nil
			},
		},
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != command.RunResult {
		t.Fatalf("bad: %d", exitCode)
	}

	if !command.RunCalled {
		t.Fatalf("run should be called")
	}

	if !reflect.DeepEqual(command.RunArgs, []string{"-bar", "-baz"}) {
		t.Fatalf("bad args: %#v", command.RunArgs)
	}
}

func TestCLIRun_prefix(t *testing.T) {
	buf := new(bytes.Buffer)
	command := new(MockCommand)
	cli := &CLI{
		Args: []string{"foobar"},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return command, nil
			},

			"foo bar": func() (Command, error) {
				return command, nil
			},
		},
		HelpWriter: buf,
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != 1 {
		t.Fatalf("bad: %d", exitCode)
	}

	if command.RunCalled {
		t.Fatalf("run should not be called")
	}
}

func TestCLIRun_default(t *testing.T) {
	commandBar := new(MockCommand)
	commandBar.RunResult = 42

	cli := &CLI{
		Args: []string{"-bar", "-baz"},
		Commands: map[string]CommandFactory{
			"": func() (Command, error) {
				return commandBar, nil
			},
			"foo": func() (Command, error) {
				return new(MockCommand), nil
			},
		},
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != commandBar.RunResult {
		t.Fatalf("bad: %d", exitCode)
	}

	if !commandBar.RunCalled {
		t.Fatalf("run should be called")
	}

	if !reflect.DeepEqual(commandBar.RunArgs, []string{"-bar", "-baz"}) {
		t.Fatalf("bad args: %#v", commandBar.RunArgs)
	}
}

func TestCLIRun_helpNested(t *testing.T) {
	helpCalled := false
	buf := new(bytes.Buffer)
	cli := &CLI{
		Args: []string{"--help"},
		Commands: map[string]CommandFactory{
			"foo sub42": func() (Command, error) {
				return new(MockCommand), nil
			},
		},
		HelpFunc: func(m map[string]CommandFactory) string {
			helpCalled = true

			var keys []string
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			expected := []string{"foo"}
			if !reflect.DeepEqual(keys, expected) {
				return fmt.Sprintf("error: contained sub: %#v", keys)
			}

			return ""
		},
		HelpWriter: buf,
	}

	code, err := cli.Run()
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	if code != 0 {
		t.Fatalf("Code: %d", code)
	}

	if !helpCalled {
		t.Fatal("help not called")
	}
}

func TestCLIRun_nested(t *testing.T) {
	command := new(MockCommand)
	cli := &CLI{
		Args: []string{"foo", "bar", "-bar", "-baz"},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return new(MockCommand), nil
			},
			"foo bar": func() (Command, error) {
				return command, nil
			},
		},
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != command.RunResult {
		t.Fatalf("bad: %d", exitCode)
	}

	if !command.RunCalled {
		t.Fatalf("run should be called")
	}

	if !reflect.DeepEqual(command.RunArgs, []string{"-bar", "-baz"}) {
		t.Fatalf("bad args: %#v", command.RunArgs)
	}
}

func TestCLIRun_nestedMissingParent(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := &CLI{
		Args: []string{"foo"},
		Commands: map[string]CommandFactory{
			"foo bar": func() (Command, error) {
				return &MockCommand{SynopsisText: "hi!"}, nil
			},
		},
		HelpWriter: buf,
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != 1 {
		t.Fatalf("bad exit code: %d", exitCode)
	}

	if buf.String() != testCommandNestedMissingParent {
		t.Fatalf("bad: %#v", buf.String())
	}
}

func TestCLIRun_printHelp(t *testing.T) {
	testCases := [][]string{
		{"-h"},
		{"--help"},
	}

	for _, testCase := range testCases {
		buf := new(bytes.Buffer)
		helpText := "foo"

		cli := &CLI{
			Args: testCase,
			Commands: map[string]CommandFactory{
				"foo": func() (Command, error) {
					return new(MockCommand), nil
				},
			},
			HelpFunc: func(map[string]CommandFactory) string {
				return helpText
			},
			HelpWriter: buf,
		}

		code, err := cli.Run()
		if err != nil {
			t.Errorf("Args: %#v. Error: %s", testCase, err)
			continue
		}

		if code != 0 {
			t.Errorf("Args: %#v. Code: %d", testCase, code)
			continue
		}

		if !strings.Contains(buf.String(), helpText) {
			t.Errorf("Args: %#v. Text: %v", testCase, buf.String())
		}
	}
}

func TestCLIRun_printHelpIllegal(t *testing.T) {
	testCases := [][]string{
		{},
		{"i-dont-exist"},
		{"-bad-flag", "foo"},
	}

	for _, testCase := range testCases {
		buf := new(bytes.Buffer)
		helpText := "foo"

		cli := &CLI{
			Args: testCase,
			Commands: map[string]CommandFactory{
				"foo": func() (Command, error) {
					return &MockCommand{HelpText: helpText}, nil
				},
				"foo sub42": func() (Command, error) {
					return new(MockCommand), nil
				},
			},
			HelpFunc: func(m map[string]CommandFactory) string {
				var keys []string
				for k := range m {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				expected := []string{"foo"}
				if !reflect.DeepEqual(keys, expected) {
					return fmt.Sprintf("error: contained sub: %#v", keys)
				}

				return helpText
			},
			HelpWriter: buf,
		}

		code, err := cli.Run()
		if err != nil {
			t.Errorf("Args: %#v. Error: %s", testCase, err)
			continue
		}

		if code != 1 {
			t.Errorf("Args: %#v. Code: %d", testCase, code)
			continue
		}

		if strings.Contains(buf.String(), "error") {
			t.Errorf("Args: %#v. Text: %v", testCase, buf.String())
		}

		if !strings.Contains(buf.String(), helpText) {
			t.Errorf("Args: %#v. Text: %v", testCase, buf.String())
		}
	}
}

func TestCLIRun_printCommandHelp(t *testing.T) {
	testCases := [][]string{
		{"--help", "foo"},
		{"-h", "foo"},
	}

	for _, args := range testCases {
		command := &MockCommand{
			HelpText: "donuts",
		}

		buf := new(bytes.Buffer)
		cli := &CLI{
			Args: args,
			Commands: map[string]CommandFactory{
				"foo": func() (Command, error) {
					return command, nil
				},
			},
			HelpWriter: buf,
		}

		exitCode, err := cli.Run()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if exitCode != 0 {
			t.Fatalf("bad exit code: %d", exitCode)
		}

		if buf.String() != (command.HelpText + "\n") {
			t.Fatalf("bad: %#v", buf.String())
		}
	}
}

func TestCLIRun_printCommandHelpNested(t *testing.T) {
	testCases := [][]string{
		{"--help", "foo", "bar"},
		{"-h", "foo", "bar"},
	}

	for _, args := range testCases {
		command := &MockCommand{
			HelpText: "donuts",
		}

		buf := new(bytes.Buffer)
		cli := &CLI{
			Args: args,
			Commands: map[string]CommandFactory{
				"foo bar": func() (Command, error) {
					return command, nil
				},
			},
			HelpWriter: buf,
		}

		exitCode, err := cli.Run()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if exitCode != 0 {
			t.Fatalf("bad exit code: %d", exitCode)
		}

		if buf.String() != (command.HelpText + "\n") {
			t.Fatalf("bad: %#v", buf.String())
		}
	}
}

func TestCLIRun_printCommandHelpSubcommands(t *testing.T) {
	testCases := [][]string{
		{"--help", "foo"},
		{"-h", "foo"},
	}

	for _, args := range testCases {
		command := &MockCommand{
			HelpText: "donuts",
		}

		buf := new(bytes.Buffer)
		cli := &CLI{
			Args: args,
			Commands: map[string]CommandFactory{
				"foo": func() (Command, error) {
					return command, nil
				},
				"foo bar": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
				"foo zip": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
				"foo zap": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
				"foo banana": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
				"foo longer": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
				"foo longer longest": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
			},
			HelpWriter: buf,
		}

		exitCode, err := cli.Run()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if exitCode != 0 {
			t.Fatalf("bad exit code: %d", exitCode)
		}

		if buf.String() != testCommandHelpSubcommandsOutput {
			t.Fatalf("bad: %#v\n\n'%#v'\n\n'%#v'", args, buf.String(), testCommandHelpSubcommandsOutput)
		}
	}
}

func TestCLIRun_printCommandHelpSubcommandsNestedTwoLevel(t *testing.T) {
	testCases := [][]string{
		{"--help", "L1"},
		{"-h", "L1"},
	}

	for _, args := range testCases {
		command := &MockCommand{
			HelpText: "donuts",
		}

		buf := new(bytes.Buffer)
		cli := &CLI{
			Args: args,
			Commands: map[string]CommandFactory{
				"L1": func() (Command, error) {
					return command, nil
				},
				"L1 L2A": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
				"L1 L2B": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
				"L1 L2A L3A": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
				"L1 L2A L3B": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
			},
			HelpWriter: buf,
		}

		exitCode, err := cli.Run()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if exitCode != 0 {
			t.Fatalf("bad exit code: %d", exitCode)
		}

		if buf.String() != testCommandHelpSubcommandsTwoLevelOutput {
			t.Fatalf("bad: %#v\n\n%s\n\n%s", args, buf.String(), testCommandHelpSubcommandsOutput)
		}
	}
}

func TestCLIRun_printCommandHelpTemplate(t *testing.T) {
	testCases := [][]string{
		{"--help", "foo"},
		{"-h", "foo"},
	}

	for _, args := range testCases {
		command := &MockCommandHelpTemplate{
			MockCommand: MockCommand{
				HelpText: "donuts",
			},

			HelpTemplateText: "hello {{.Help}}",
		}

		buf := new(bytes.Buffer)
		cli := &CLI{
			Args: args,
			Commands: map[string]CommandFactory{
				"foo": func() (Command, error) {
					return command, nil
				},
			},
			HelpWriter: buf,
		}

		exitCode, err := cli.Run()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if exitCode != 0 {
			t.Fatalf("bad exit code: %d", exitCode)
		}

		if buf.String() != "hello "+command.HelpText+"\n" {
			t.Fatalf("bad: %#v", buf.String())
		}
	}
}

func TestCLIRun_autocompleteBoth(t *testing.T) {
	command := new(MockCommand)
	cli := &CLI{
		Args: []string{
			"-" + defaultAutocompleteInstall,
			"-" + defaultAutocompleteUninstall,
		},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return command, nil
			},
		},

		Name:                  "foo",
		Autocomplete:          true,
		autocompleteInstaller: &mockAutocompleteInstaller{},
	}

	exitCode, err := cli.Run()
	if err == nil {
		t.Fatal("should error")
	}

	if exitCode != 1 {
		t.Fatalf("bad: %d", exitCode)
	}

	if command.RunCalled {
		t.Fatalf("run should not be called")
	}
}

func TestCLIRun_autocompleteInstall(t *testing.T) {
	command := new(MockCommand)
	installer := new(mockAutocompleteInstaller)
	cli := &CLI{
		Args: []string{
			"-" + defaultAutocompleteInstall,
		},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return command, nil
			},
		},

		Name:                  "foo",
		Autocomplete:          true,
		autocompleteInstaller: installer,
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != 0 {
		t.Fatalf("bad: %d", exitCode)
	}

	if command.RunCalled {
		t.Fatalf("run should not be called")
	}

	if !installer.InstallCalled {
		t.Fatal("should call install")
	}
}

func TestCLIRun_autocompleteUninstall(t *testing.T) {
	command := new(MockCommand)
	installer := new(mockAutocompleteInstaller)
	cli := &CLI{
		Args: []string{
			"-" + defaultAutocompleteUninstall,
		},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return command, nil
			},
		},

		Name:                  "foo",
		Autocomplete:          true,
		autocompleteInstaller: installer,
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != 0 {
		t.Fatalf("bad: %d", exitCode)
	}

	if command.RunCalled {
		t.Fatalf("run should not be called")
	}

	if !installer.UninstallCalled {
		t.Fatal("should call uninstall")
	}
}

func TestCLIRun_autocompleteNoName(t *testing.T) {
	command := new(MockCommand)
	installer := new(mockAutocompleteInstaller)
	cli := &CLI{
		Args: []string{"foo"},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return command, nil
			},
		},

		Autocomplete:          true,
		autocompleteInstaller: installer,
	}

	exitCode, err := cli.Run()
	if err == nil {
		t.Fatal("should error")
	}

	if exitCode != 1 {
		t.Fatalf("bad: %d", exitCode)
	}

	if command.RunCalled {
		t.Fatalf("run should not be called")
	}
}

func TestCLIAutocomplete_root(t *testing.T) {
	cases := []struct {
		Completed []string
		Last      string
		Expected  []string
	}{
		{nil, "-v", []string{"-version"}},
		{nil, "-h", []string{"-help"}},
		{nil, "-a", []string{
			"-" + defaultAutocompleteInstall,
			"-" + defaultAutocompleteUninstall,
		}},

		{nil, "f", []string{"foo"}},
		{nil, "n", []string{"nodes", "noodles"}},
		{nil, "noo", []string{"noodles"}},
		{nil, "su", []string{"sub"}},

		// Make sure global flags work on subcommands
		{[]string{"sub"}, "-v", []string{"-version"}},
		{[]string{"sub"}, "o", []string{"one"}},
	}

	for _, tc := range cases {
		t.Run(tc.Last, func(t *testing.T) {
			command := new(MockCommand)
			cli := &CLI{
				Commands: map[string]CommandFactory{
					"foo":     func() (Command, error) { return command, nil },
					"nodes":   func() (Command, error) { return command, nil },
					"noodles": func() (Command, error) { return command, nil },
					"sub one": func() (Command, error) { return command, nil },
					"sub two": func() (Command, error) { return command, nil },
				},

				Autocomplete: true,
			}

			// Initialize
			cli.init()

			// Test the autocompleter
			actual := cli.autocomplete.Command.Predict(complete.Args{
				Completed: tc.Completed,
				Last:      tc.Last,
			})
			sort.Strings(actual)

			if !reflect.DeepEqual(actual, tc.Expected) {
				t.Fatalf("bad prediction: %#v", actual)
			}
		})
	}
}

func TestCLIAutocomplete_rootGlobalFlags(t *testing.T) {
	cases := []struct {
		Completed []string
		Last      string
		Expected  []string
	}{
		{nil, "-v", []string{"-version"}},
		{nil, "-t", []string{"-tubes"}},
	}

	for _, tc := range cases {
		t.Run(tc.Last, func(t *testing.T) {
			command := new(MockCommand)
			cli := &CLI{
				Commands: map[string]CommandFactory{
					"foo": func() (Command, error) { return command, nil },
				},

				Autocomplete: true,
				AutocompleteGlobalFlags: map[string]complete.Predictor{
					"-tubes": complete.PredictNothing,
				},
			}

			// Initialize
			cli.init()

			// Test the autocompleter
			actual := cli.autocomplete.Command.Predict(complete.Args{
				Completed: tc.Completed,
				Last:      tc.Last,
			})
			sort.Strings(actual)

			if !reflect.DeepEqual(actual, tc.Expected) {
				t.Fatalf("bad prediction: %#v", actual)
			}
		})
	}
}

func TestCLIAutocomplete_subcommandArgs(t *testing.T) {
	cases := []struct {
		Completed []string
		Last      string
		Expected  []string
	}{
		{[]string{"foo"}, "RE", []string{"README.md"}},
		{[]string{"foo", "-go"}, "asdf", []string{"yo"}},
	}

	for _, tc := range cases {
		t.Run(tc.Last, func(t *testing.T) {
			command := new(MockCommandAutocomplete)
			command.AutocompleteArgsValue = complete.PredictFiles("*")
			command.AutocompleteFlagsValue = map[string]complete.Predictor{
				"-go": complete.PredictFunc(func(complete.Args) []string {
					return []string{"yo"}
				}),
			}

			cli := &CLI{
				Commands: map[string]CommandFactory{
					"foo": func() (Command, error) {
						return command, nil
					},
				},

				Autocomplete: true,
			}

			// Initialize
			cli.init()

			// Test the autocompleter
			actual := cli.autocomplete.Command.Predict(complete.Args{
				Completed:     tc.Completed,
				Last:          tc.Last,
				LastCompleted: tc.Completed[len(tc.Completed)-1],
			})
			sort.Strings(actual)

			if !reflect.DeepEqual(actual, tc.Expected) {
				t.Fatalf("bad prediction: %#v", actual)
			}
		})
	}
}

func TestCLISubcommand(t *testing.T) {
	testCases := []struct {
		args       []string
		subcommand string
	}{
		{[]string{"bar"}, "bar"},
		{[]string{"foo", "-h"}, "foo"},
		{[]string{"-h", "bar"}, "bar"},
		{[]string{"foo", "bar", "-h"}, "foo"},
	}

	for _, testCase := range testCases {
		cli := &CLI{Args: testCase.args}
		result := cli.Subcommand()

		if result != testCase.subcommand {
			t.Errorf("Expected %#v, got %#v. Args: %#v",
				testCase.subcommand, result, testCase.args)
		}
	}
}

func TestCLISubcommand_nested(t *testing.T) {
	testCases := []struct {
		args       []string
		subcommand string
	}{
		{[]string{"bar"}, "bar"},
		{[]string{"foo", "-h"}, "foo"},
		{[]string{"-h", "bar"}, "bar"},
		{[]string{"foo", "bar", "-h"}, "foo bar"},
		{[]string{"foo", "bar", "baz", "-h"}, "foo bar"},
		{[]string{"foo", "bar", "-h", "baz"}, "foo bar"},
		{[]string{"-h", "foo", "bar"}, "foo bar"},
	}

	for _, testCase := range testCases {
		cli := &CLI{
			Args: testCase.args,
			Commands: map[string]CommandFactory{
				"foo bar": func() (Command, error) {
					return new(MockCommand), nil
				},
			},
		}
		result := cli.Subcommand()

		if result != testCase.subcommand {
			t.Errorf("Expected %#v, got %#v. Args: %#v",
				testCase.subcommand, result, testCase.args)
		}
	}
}

const testCommandNestedMissingParent = `This command is accessed by using one of the subcommands below.

Subcommands:
    bar    hi!
`

const testCommandHelpSubcommandsOutput = `donuts

Subcommands:
    banana    hi!
    bar       hi!
    longer    hi!
    zap       hi!
    zip       hi!
`

const testCommandHelpSubcommandsTwoLevelOutput = `donuts

Subcommands:
    L2A    hi!
    L2B    hi!
`
