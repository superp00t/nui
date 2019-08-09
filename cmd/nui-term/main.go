package main

import (
	"fmt"

	"github.com/c-bata/go-prompt"
	"github.com/superp00t/nui/term"
)

var LivePrefixState struct {
	LivePrefix string
	IsEnable   bool
}

func executor(in string) {
	fmt.Println("Your input: " + in)
	if in == "" {
		LivePrefixState.IsEnable = false
		LivePrefixState.LivePrefix = in
		return
	}
	LivePrefixState.LivePrefix = in + "> "
	LivePrefixState.IsEnable = true
}

func completer(in prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "users", Description: "Store the username and age"},
		{Text: "articles", Description: "Store the article text posted by user"},
		{Text: "comments", Description: "Store the text commented to articles"},
		{Text: "groups", Description: "Combine users with specific rules"},
	}
	return prompt.FilterHasPrefix(s, in.GetWordBeforeCursor(), true)
}

func changeLivePrefix() (string, bool) {
	return LivePrefixState.LivePrefix, LivePrefixState.IsEnable
}

func main() {
	t := term.New("test term")

	t.win.OnLoad

	go func() {
		p := prompt.New(
			executor,
			completer,
			prompt.OptionPrefix(">>> "),
			prompt.OptionLivePrefix(changeLivePrefix),
			prompt.OptionTitle("live-prefix-example"),
			prompt.OptionParser(t),
			prompt.OptionWriter(t))

		p.Run()
	}()

	panic(t.Run())
}
