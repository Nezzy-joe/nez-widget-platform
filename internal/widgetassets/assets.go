package widgetassets

import "embed"

// Files contains the static assets shipped with the API binary.
//
//go:embed widget/v1/widget.js
var Files embed.FS
