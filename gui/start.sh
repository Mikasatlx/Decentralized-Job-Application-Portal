#!/bin/bash

gnome-terminal -- go run mod.go start --configpath ./config_0.json &

gnome-terminal -- go run mod.go start --configpath ./config_1.json &

gnome-terminal -- go run mod.go start --configpath ./config_2.json



# xterm -title "App 1" -hold -e "go run mod.go start --configpath ./config_0.json" &
# xterm -title "App 2" -hold -e "go run mod.go start --configpath ./config_1.json" &
# xterm -title "App 3" -hold -e "go run mod.go start --configpath ./config_2.json"