// Public Domain (-) 2010-2018 The Golly Authors.
// See the Golly UNLICENSE file for details.

// Package optparse provides utility functions for the parsing and
// autocompletion of command line arguments.
package optparse

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/flynn/go-shlex"
	"github.com/tav/golly/log"
	"github.com/tav/golly/process"
	"github.com/tav/golly/structure"
)

type valueType int

const (
	boolValue valueType = iota
	durationValue
	floatValue
	intValue
	intSliceValue
	stringValue
	stringSliceValue
)

type Completer interface {
	Complete([]string, int) []string
}

type IndexedCompleter map[int][]string

func (c IndexedCompleter) Complete(args []string, cword int) []string {
	return c[cword]
}

type ListCompleter []string

func (c ListCompleter) Complete([]string, int) []string {
	return c
}

func exit(message string, v ...interface{}) {
	log.Errorf(message, v...)
	process.Exit(1)
}

type Parser struct {
	Completer             Completer
	HelpOptDescription    string
	HideHelpOpt           bool
	HideVersionOpt        bool
	OptPadding            int
	ParseHelp             bool
	ParseVersion          bool
	Usage                 string
	VersionOptDescription string
	haltFlagParsing       bool
	haltFlagParsingN      int
	haltFlagParsingString string
	helpAdded             bool
	long2options          map[string]*option
	longFlags             []string
	nextCompleter         Completer
	nextFlags             []string
	nextHidden            bool
	nextLabel             string
	nextRequired          bool
	options               []*option
	short2options         map[string]*option
	shortFlags            []string
	version               func() string
	versionAdded          bool
}

type option struct {
	completer   Completer
	defined     bool
	description string
	hidden      bool
	label       string
	longFlag    string
	required    bool
	shortFlag   string
	value       interface{}
	valueType   valueType
}

func (op *option) FlagString() string {
	output := "  "
	if op.shortFlag != "" {
		output += op.shortFlag
		if op.longFlag != "" {
			output += ", "
		}
	}
	if op.longFlag != "" {
		output += op.longFlag
	}
	if op.label != "" {
		output += " " + op.label
	}
	return output
}

func (op *option) Print(format string) {
	flagString := op.FlagString()
	fmt.Printf(format, flagString, op.description)
}

func (p *Parser) newOpt(description string, showLabel bool) *option {
	op := &option{}
	op.completer = p.nextCompleter
	op.description = description
	op.hidden = p.nextHidden
	op.required = p.nextRequired
	for _, flag := range p.nextFlags {
		if strings.HasPrefix(flag, "--") {
			op.longFlag = flag
			p.long2options[flag] = op
			p.longFlags = append(p.longFlags, flag)
		} else if strings.HasPrefix(flag, "-") {
			op.shortFlag = flag
			p.short2options[flag] = op
			p.shortFlags = append(p.shortFlags, flag)
		}
	}
	if op.shortFlag == "" && op.longFlag == "" {
		exit("optparse: no -short or --long flags found for option with description: %s", description)
	}
	if !op.hidden {
		if showLabel {
			if p.nextLabel != "" {
				op.label = p.nextLabel
			} else {
				if op.longFlag != "" {
					op.label = strings.ToUpper(strings.TrimLeft(op.longFlag, "-"))
				} else {
					op.label = strings.ToUpper(strings.TrimLeft(op.shortFlag, "-"))
				}
			}
		}
		width := len(op.FlagString())
		if width > p.OptPadding {
			p.OptPadding = width
		}
	}
	p.options = append(p.options, op)
	p.nextCompleter = nil
	p.nextFlags = nil
	p.nextHidden = false
	p.nextLabel = ""
	p.nextRequired = false
	return op
}

// Bool defines a new option with the given description and
// optional default value.
func (p *Parser) Bool(description string) *bool {
	v := false
	op := p.newOpt(description, false)
	op.valueType = boolValue
	op.value = &v
	return &v
}

// Duration defines a new option with the given description and optional default
// value.
func (p *Parser) Duration(description string, defaultValue ...time.Duration) *time.Duration {
	v := time.Duration(0)
	if len(defaultValue) > 0 {
		v = defaultValue[0]
	} else if strings.HasSuffix(description, "]") {
		if idx := strings.LastIndex(description, "["); idx != 1 {
			var err error
			v, err = time.ParseDuration(description[idx+1 : len(description)-1])
			if err != nil {
				exit("optparse: could not parse default value from: %s", description)
			}
		}
	}
	op := p.newOpt(description, true)
	op.valueType = durationValue
	op.value = &v
	return &v
}

// Float defines a new option with the given description and optional default
// value.
func (p *Parser) Float(description string, defaultValue ...float64) *float64 {
	var v float64
	if len(defaultValue) > 0 {
		v = defaultValue[0]
	} else if strings.HasSuffix(description, "]") {
		if idx := strings.LastIndex(description, "["); idx != 1 {
			var err error
			v, err = strconv.ParseFloat(description[idx+1:len(description)-1], 64)
			if err != nil {
				exit("optparse: could not parse default value from: %s", description)
			}
		}
	}
	op := p.newOpt(description, true)
	op.valueType = floatValue
	op.value = &v
	return &v
}

// Int defines a new option with the given description and
// optional default value.
func (p *Parser) Int(description string, defaultValue ...int) *int {
	v := 0
	if len(defaultValue) > 0 {
		v = defaultValue[0]
	} else if strings.HasSuffix(description, "]") {
		if idx := strings.LastIndex(description, "["); idx != 1 {
			var err error
			v, err = strconv.Atoi(description[idx+1 : len(description)-1])
			if err != nil {
				exit("optparse: could not parse default value from: %s", description)
			}
		}
	}
	op := p.newOpt(description, true)
	op.valueType = intValue
	op.value = &v
	return &v
}

// String defines a new option with the given description
// and optional default value.
func (p *Parser) String(description string, defaultValue ...string) *string {
	v := ""
	if len(defaultValue) > 0 {
		v = defaultValue[0]
	} else if strings.HasSuffix(description, "]") {
		if idx := strings.LastIndex(description, "["); idx != 1 {
			v = description[idx+1 : len(description)-1]
		}
	}
	op := p.newOpt(description, true)
	op.valueType = stringValue
	op.value = &v
	return &v
}

// Flags specifies the -short and/or --long flags to use for
// the next defined option.
func (p *Parser) Flags(flags ...string) *Parser {
	p.nextFlags = flags
	return p
}

// Hidden will suppress the next defined option from being
// displayed in the auto-generated usage output.
func (p *Parser) Hidden() *Parser {
	p.nextHidden = true
	return p
}

// Required indicates that the option parser should raise an
// error if the next defined option is not specified.
func (p *Parser) Required() *Parser {
	p.nextRequired = true
	return p
}

// WithOptCompleter will use the provided Completer to
// autocomplete the next defined option.
func (p *Parser) WithOptCompleter(c Completer) *Parser {
	p.nextCompleter = c
	return p
}

// Label will use the given label string for the next
// defined option.
func (p *Parser) Label(label string) *Parser {
	p.nextLabel = label
	return p
}

func (p *Parser) HaltFlagParsing(v interface{}) *Parser {
	if n, ok := v.(int); ok && n > 0 {
		p.haltFlagParsing = true
		p.haltFlagParsingN = n
	} else if s, ok := v.(string); ok && s != "" {
		p.haltFlagParsing = true
		p.haltFlagParsingString = s
	} else {
		exit("optparse: expected non-empty string or int value for HaltFlagParsing()")
	}
	return p
}

// Parse will parse the given args slice and try and define
// the defined options.
func (p *Parser) Parse(args []string) (remainder []string) {

	if p.ParseHelp && !p.helpAdded {
		description := p.HelpOptDescription
		if description == "" {
			description = "Show this help and exit"
		}
		if p.HideHelpOpt {
			p.Hidden()
		}
		p.Flags("-h", "--help").Bool(description)
		p.helpAdded = true
	}

	if p.ParseVersion && !p.versionAdded {
		description := p.VersionOptDescription
		if description == "" {
			description = "Show the version and exit"
		}
		if p.HideVersionOpt {
			p.Hidden()
		}
		p.Flags("-v", "--version").Bool(description)
		p.versionAdded = true
	}

	argLength := len(args) - 1
	complete, words, compWord, prefix := getCompletionData()

	// Command-line auto-completion support.
	if complete {

		seenLong := []string{}
		seenShort := []string{}

		subcommands, err := shlex.Split(args[0])
		if err != nil {
			process.Exit(1)
		}

		words = words[len(subcommands):]
		compWord -= len(subcommands)

		argWords := []string{}
		skipNext := false
		optCount := 0

		for _, word := range words {
			if skipNext {
				skipNext = false
				optCount += 1
				continue
			}
			if strings.HasPrefix(word, "--") && word != "--" {
				op, ok := p.long2options[word]
				if ok {
					seenLong = append(seenLong, op.longFlag)
					seenShort = append(seenShort, op.shortFlag)
					if op.label != "" {
						skipNext = true
					}
				}
				optCount += 1
			} else if strings.HasPrefix(word, "-") && !(word == "-" || word == "--") {
				op, ok := p.short2options[word]
				if ok {
					seenLong = append(seenLong, op.longFlag)
					seenShort = append(seenShort, op.shortFlag)
					if op.label != "" {
						skipNext = true
					}
				}
				optCount += 1
			} else {
				argWords = append(argWords, word)
				if p.haltFlagParsing {
					if p.haltFlagParsingString != "" {
						if word == p.haltFlagParsingString {
							process.Exit(1)
						}
					} else if (compWord - optCount) == p.haltFlagParsingN {
						process.Exit(1)
					}
				}
			}
		}

		// Pass to the shell completion if the previous word was a flag
		// expecting some parameter.
		if compWord >= 1 {
			var completer Completer
			prev := words[compWord-1]
			if prev != "--" && prev != "-" {
				if strings.HasPrefix(prev, "--") {
					op, ok := p.long2options[prev]
					if ok && op.label != "" {
						if op.completer == nil {
							process.Exit(1)
						} else {
							completer = op.completer
						}
					}
				} else if strings.HasPrefix(prev, "-") {
					op, ok := p.short2options[prev]
					if ok && op.label != "" {
						if op.completer == nil {
							process.Exit(1)
						} else {
							completer = op.completer
						}
					}
				}
			}
			if completer != nil {
				completions := make([]string, 0)
				for _, item := range completer.Complete(argWords, compWord) {
					if strings.HasPrefix(item, prefix) {
						completions = append(completions, item)
					}
				}
				fmt.Print(strings.Join(completions, " "))
				process.Exit(1)
			}
		}

		completions := make([]string, 0)

		if p.Completer != nil {
			for _, item := range p.Completer.Complete(argWords, compWord-optCount) {
				if strings.HasPrefix(item, prefix) {
					completions = append(completions, item)
				}
			}
		}

		for flag, op := range p.long2options {
			if !(contains(seenLong, op.longFlag) || contains(seenShort, op.shortFlag) || op.hidden) {
				if strings.HasPrefix(flag, prefix) {
					completions = append(completions, flag)
				}
			}
		}

		for flag, op := range p.short2options {
			if !(contains(seenLong, op.longFlag) || contains(seenShort, op.shortFlag) || op.hidden) {
				if strings.HasPrefix(flag, prefix) {
					completions = append(completions, flag)
				}
			}
		}

		fmt.Print(strings.Join(completions, " "))
		process.Exit(1)

	}

	if argLength == 0 {
		return
	}

	var op *option
	var ok bool

	idx := 1
	addNext := false

	for {
		arg := args[idx]
		noOpt := true
		if addNext {
			remainder = append(remainder, arg)
			if idx == argLength {
				break
			}
			idx += 1
			continue
		} else if strings.HasPrefix(arg, "--") && arg != "--" {
			op, ok = p.long2options[arg]
			if ok {
				noOpt = false
			}
		} else if strings.HasPrefix(arg, "-") && !(arg == "-" || arg == "--") {
			op, ok = p.short2options[arg]
			if ok {
				noOpt = false
			}
		} else {
			remainder = append(remainder, arg)
			if p.haltFlagParsing {
				if arg == p.haltFlagParsingString {
					addNext = true
				} else if len(remainder) == p.haltFlagParsingN {
					addNext = true
				}
			}
			if idx == argLength {
				break
			}
			idx += 1
			continue
		}
		if noOpt {
			exit("%s: no such option: %s", args[0], arg)
		}
		if op.label != "" {
			if idx == argLength {
				exit("%s: %s option requires an argument", args[0], arg)
			}
		}
		if op.valueType == boolValue {
			if op.longFlag == "--help" && p.ParseHelp {
				p.PrintUsage()
				process.Exit(1)
			} else if op.longFlag == "--version" && p.ParseVersion {
				fmt.Printf("%s\n", p.version())
				process.Exit(0)
			}
			v := op.value.(*bool)
			*v = true
			op.defined = true
			idx += 1
		} else if op.valueType == durationValue {
			if idx == argLength {
				exit("%s: no value specified for %s", args[0], arg)
			}
			durationValue, err := time.ParseDuration(args[idx+1])
			if err != nil {
				exit("%s: couldn't convert %s value '%s' to a duration", args[0], arg, args[idx+1])
			}
			v := op.value.(*time.Duration)
			*v = durationValue
			op.defined = true
			idx += 2
		} else if op.valueType == stringValue {
			if idx == argLength {
				exit("%s: no value specified for %s", args[0], arg)
			}
			v := op.value.(*string)
			*v = args[idx+1]
			op.defined = true
			idx += 2
		} else if op.valueType == intValue {
			if idx == argLength {
				exit("%s: no value specified for %s", args[0], arg)
			}
			intValue, err := strconv.Atoi(args[idx+1])
			if err != nil {
				exit("%s: couldn't convert %s value '%s' to an integer", args[0], arg, args[idx+1])
			}
			v := op.value.(*int)
			*v = intValue
			op.defined = true
			idx += 2
		} else if op.valueType == floatValue {
			if idx == argLength {
				exit("%s: no value specified for %s", args[0], arg)
			}
			floatValue, err := strconv.ParseFloat(args[idx+1], 64)
			if err != nil {
				exit("%s: couldn't convert %s value '%s' to a float", args[0], arg, args[idx+1])
			}
			v := op.value.(*float64)
			*v = floatValue
			op.defined = true
			idx += 2
		}
		if idx > argLength {
			break
		}
	}

	for _, op := range p.options {
		if op.required && !op.defined {
			exit("%s: required: %s", args[0], op)
		}
	}

	return

}

// PrintUsage generates and prints a default help usage output.
func (p *Parser) PrintUsage() {
	fmt.Print(p.Usage)
	optFormat := fmt.Sprintf("%%-%ds%%s\n", p.OptPadding+4)
	printHeader := true
	for _, op := range p.options {
		if !op.hidden {
			if printHeader {
				fmt.Print("\nOptions:\n")
				printHeader = false
			}
			op.Print(optFormat)
		}
	}
}

// SetVersion lets you specify a version string or function
// returning a string for use by the version option handler.
func (p *Parser) SetVersion(value interface{}) *Parser {
	var setVersion bool
	var versionFunc func() string
	if versionString, found := value.(string); found {
		if len(versionString) != 0 {
			setVersion = true
			versionFunc = func() string {
				return versionString
			}
		}
	} else if versionFunc, found = value.(func() string); found {
		setVersion = true
	}
	if !setVersion {
		exit("optparse: the SetVersion value needs to be a string or a function returning a string")
	}
	p.version = versionFunc
	p.ParseVersion = true
	return p
}

// New returns a fresh parser with the given usage header
// and optional version string.
func New(usage string) *Parser {
	p := &Parser{}
	p.long2options = make(map[string]*option)
	p.short2options = make(map[string]*option)
	p.OptPadding = 20
	p.ParseHelp = true
	p.Usage = usage
	return p
}

func contains(list []string, item string) bool {
	for _, elem := range list {
		if elem == item {
			return true
		}
	}
	return false
}

func debug(filename, message string, v ...interface{}) {
	f, _ := os.Create(filename + ".txt")
	fmt.Fprintf(f, message, v...)
	f.Close()
}

func getCompletionData() (complete bool, words []string, compWord int, prefix string) {

	var err error

	autocomplete := os.Getenv("OPTPARSE_AUTO_COMPLETE")
	if autocomplete != "" {

		complete = true

		words, err = shlex.Split(os.Getenv("COMP_LINE"))
		if err != nil {
			exit("optparse: could not shlex autocompletion words: %s", err)
		}

		compWord, err = strconv.Atoi(os.Getenv("COMP_CWORD"))
		if err != nil {
			process.Exit(1)
		}

		if compWord > 0 {
			if compWord < len(words) {
				prefix = words[compWord]
			}
		}

	}

	return

}

// Commands provides support for git subcommands style command handling.
func Commands(name string, version interface{}, commands map[string]func([]string, string), commandsUsage map[string]string, additional ...string) {

	var commandNames, helpCommands []string
	var complete bool
	var mainUsage string

	callCommand := func(command string, args []string, complete bool) {
		var findexe bool
		if command[0] == '-' {
			args[0] = name
		} else {
			args[0] = fmt.Sprintf("%s %s", name, command)
			findexe = true
		}
		if handler, ok := commands[command]; ok {
			handler(args, commandsUsage[command])
		} else if findexe {

			exe := fmt.Sprintf("%s-%s", strings.Replace(name, " ", "-", -1), command)
			exePath, err := exec.LookPath(exe)
			if err != nil {
				exit("%s: '%s' is not a valid command: see '%s help'", name, command, name)
			}

			args[0] = exe
			process, err := os.StartProcess(exePath, args,
				&os.ProcAttr{
					Dir:   ".",
					Env:   os.Environ(),
					Files: []*os.File{nil, os.Stdout, os.Stderr},
				})

			if err != nil {
				exit("%s: %s", exe, err)
			}

			_, err = process.Wait()
			if err != nil {
				exit("%s: %s", exe, err)
			}

		} else {
			exit(fmt.Sprintf("%s: no such option: %s", name, command))
		}
		process.Exit(0)
	}

	if _, ok := commands["help"]; !ok {
		commands["help"] = func(args []string, usage string) {

			opts := New(mainUsage)
			opts.ParseHelp = false
			opts.Completer = ListCompleter(helpCommands)
			helpArgs := opts.Parse(args)

			if len(helpArgs) == 0 {
				fmt.Print(mainUsage)
				process.Exit(1)
			}

			if len(helpArgs) != 1 {
				exit("%s: invalid help option: '%s'", name, strings.Join(helpArgs, " "))
			}

			command := helpArgs[0]
			if command == "help" {
				exit("%s: invalid help command on help", name)
			} else {
				if !complete {
					argLen := len(os.Args)
					os.Args[argLen-2], os.Args[argLen-1] = os.Args[argLen-1], "--help"
				}
				callCommand(command, []string{name, "--help"}, false)
			}

			process.Exit(1)

		}
		commands["-h"] = commands["help"]
		commands["--help"] = commands["help"]
	}

	var setVersion bool
	var versionFunc func() string

	if versionString, found := version.(string); found {
		if len(versionString) != 0 {
			setVersion = true
			versionFunc = func() string {
				return versionString
			}
		}
	} else if versionFunc, found = version.(func() string); found {
		setVersion = true
	}

	if _, ok := commands["version"]; !ok && setVersion {
		commands["version"] = func(args []string, usage string) {
			if usage == "" {
				usage = fmt.Sprintf("  Show the %s version information.", name)
			}
			opts := New(fmt.Sprintf("Usage: %s version\n\n%s\n", name, usage))
			opts.HideHelpOpt = true
			opts.Parse(args)
			fmt.Printf("%s\n", versionFunc())
			return
		}
		commands["-v"] = commands["version"]
		commands["--version"] = commands["version"]
	}

	commandNames = make([]string, len(commands))
	helpCommands = make([]string, len(commands))
	i, j := 0, 0

	for name, _ := range commands {
		if !strings.HasPrefix(name, "-") {
			commandNames[i] = name
			i += 1
			if name != "help" {
				helpCommands[j] = name
				j += 1
			}
		}
	}

	usageKeys := structure.SortedKeys(commandsUsage)
	padding := 10

	for _, key := range usageKeys {
		if len(key) > padding {
			padding = len(key)
		}
	}

	var prefix string
	var suffix string

	lenExtra := len(additional)
	if lenExtra >= 1 {
		prefix = additional[0] + "\n\n"
	}
	if lenExtra >= 2 {
		suffix = additional[1] + "\n"
	}
	if lenExtra >= 3 {
		mainUsage = additional[2] + "\n"
	}

	mainUsage += fmt.Sprintf("Usage: %s COMMAND [OPTIONS]\n\n%sCommands:\n\n", name, prefix)
	usageLine := fmt.Sprintf("    %%-%ds  %%s\n", padding)

	for _, key := range usageKeys {
		mainUsage += fmt.Sprintf(usageLine, key, commandsUsage[key])
	}

	mainUsage += fmt.Sprintf(
		`
Run "%s help <command>" for more info on a specific command.
%s`, name, suffix)

	complete, words, compWord, prefix := getCompletionData()
	baseLength := len(strings.Split(name, " "))
	args := os.Args

	if complete && len(args) == 1 {
		if compWord == baseLength {
			completions := make([]string, 0)
			for _, cmd := range commandNames {
				if strings.HasPrefix(cmd, prefix) {
					completions = append(completions, cmd)
				}
			}
			fmt.Print(strings.Join(completions, " "))
			process.Exit(1)
		} else {
			command := words[baseLength]
			args = []string{name}
			callCommand(command, args, true)
		}
	}

	args = args[baseLength:]

	if len(args) == 0 {
		fmt.Print(mainUsage)
		process.Exit(1)
	}

	command := args[0]
	args[0] = name

	callCommand(command, args, false)

}
