package assets

import _ "embed"

// Application icon.
//
//go:embed icons/app/app.png
var AppIconPNG []byte

// Tray icons.
//
//go:embed icons/tray/online.png
var TrayOnlinePNG []byte

//go:embed icons/tray/onbatt.png
var TrayOnBattPNG []byte

//go:embed icons/tray/disconnected.png
var TrayDisconnectedPNG []byte

//go:embed icons/tray/paused.png
var TrayPausedPNG []byte

//go:embed icons/app/main-online.png
var MainOnlinePNG []byte

//go:embed icons/app/main-onbatt.png
var MainOnBattPNG []byte

//go:embed icons/app/main-disconnected.png
var MainDisconnectedPNG []byte

//go:embed icons/app/main-paused.png
var MainPausedPNG []byte
