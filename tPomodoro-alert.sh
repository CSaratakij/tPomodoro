#!/bin/sh

# Alert - Notification
header="tPomodoro"
stateLabel=""

alert_timeout=30000
if [ -n "${tPOMODORO_ALERT_TIMEOUT+x}" ]; then
	alert_timeout=$tPOMODORO_ALERT_TIMEOUT
fi

if [ "$1" = "StateFocus" ]; then
	stateLabel="Focus"
elif [ "$1" = "StateBreak" ]; then
	stateLabel="Break"
elif [ "$1" = "StateLongBreak" ]; then
	stateLabel="Long Break"
fi
subtitle="[ $stateLabel ] : times up"
notify-send -u normal -t $alert_timeout "$header" "$subtitle"

# Alert - Sound
if [ -n "${tPOMODORO_ALERT_AUDIO_PATH+x}" ]; then
	aplay "$tPOMODORO_ALERT_AUDIO_PATH" &
fi
