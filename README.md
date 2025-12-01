# tPomodoro
simple Pomodoro (TUI)

![A TUI preview](res/preview.png "")

### Features
- [x] simple pomodoro in terminal
- [x] minimal by design
- [-] customization through cli

### Installation
#### Pre-built
TODO

#### Build from source
TODO

### Keybinding
|Key|Action|
|---|------|
|s, spacebar|start / pause|
|r, reset|start / pause|
|b|next state|
|tab|toggle hint style|

### For alert when timer finish
no built-in alert, use [external scripts](tPomodoro-alert.sh) instead.<br>
include with default [beep sound](res/beep_success.wav).<br>
set environment variables appropriately.<br>

|Variable|Description|Default|
|--------|-----------|-------|
|tPOMODORO_ALERT_AUDIO_PATH|absolute path of .wav audio file to play during alert|""|
|tPOMODORO_ALERT_SCRIPT|absoulte path to alert shell scripts|""|
|tPOMODORO_ALERT_TIMEOUT|alert timeout in miliseconds|30000|

example:<br>

    $ export tPOMODORO_ALERT_TIMEOUT=0      # disable timeout
