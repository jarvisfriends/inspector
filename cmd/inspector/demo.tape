# VHS demo for cmd/inspector — regenerate from the repo root:
#   GOOS=linux GOARCH=amd64 go build -o cmd/inspector/demo-bin ./cmd/inspector
#   docker run --rm -v "$PWD:/vhs" -w /vhs ghcr.io/charmbracelet/vhs cmd/inspector/demo.tape
Set WindowBar Colorful
Output cmd/inspector/demo.gif
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 24
Set Width 1500
Set Height 900
Set TypingSpeed 60ms

Type "./cmd/inspector/demo-bin"
Enter
Sleep 2s

# Walk every inspector tab group.
Type "1"
Sleep 900ms
Type "2"
Sleep 900ms
Type "3"
Sleep 900ms
Type "4"
Sleep 900ms
Type "5"
Sleep 900ms
Type "6"
Sleep 900ms
Type "7"
Sleep 1200ms

# Exercise keyboard tab-switch flow and a little scrolling.
Tab
Sleep 800ms
Tab
Sleep 800ms
Shift+Tab
Sleep 800ms
Down
Sleep 600ms
Down
Sleep 600ms
Up
Sleep 800ms

# Loop-friendly return to the first tab.
Type "1"
Sleep 1400ms
