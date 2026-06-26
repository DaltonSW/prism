Output "./assets/filter.gif"

Require echo

Set Shell "bash"
Set FontSize 17
Set Width 800
Set Height 1000


Type "prism ./internal" Sleep 500ms  Enter

Sleep 5s

Type 'prism -F "SplitBenchmarkName" ./internal' Sleep 500ms  Enter

Sleep 5s
