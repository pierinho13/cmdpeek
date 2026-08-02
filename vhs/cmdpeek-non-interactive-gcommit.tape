Output assets/cmdpeek-non-interactive-gcommit.gif

Set Shell "zsh"
Set FontSize 18
Set Width 1800
Set Height 800
Set TypingSpeed 60ms
Set Theme "Dracula"

Require cmdpeek

Type "clear"
Enter
Sleep 500ms

Type "Run a named workflow without opening the interactive command catalog."
Ctrl+C
Sleep 1s

Type "cmdpeek --no-interactive --name gcommit  --set 'message=Add non-interactive execution demo' --yes"
Enter
Sleep 4s
