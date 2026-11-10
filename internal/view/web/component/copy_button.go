package component

import (
	nodx "github.com/nodxdev/nodxgo"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

// CopyButtonSm is a small copy button.
func CopyButtonSm(textToCopy string) nodx.Node {
	return copyButton(copyButtonProps{
		TextToCopy: textToCopy,
		Size:       SizeSm,
	})
}

// CopyButtonMd is a medium copy button.
func CopyButtonMd(textToCopy string) nodx.Node {
	return copyButton(copyButtonProps{
		TextToCopy: textToCopy,
	})
}

// CopyButtonLg is a large copy button.
func CopyButtonLg(textToCopy string) nodx.Node {
	return copyButton(copyButtonProps{
		TextToCopy: textToCopy,
		Size:       SizeLg,
	})
}

// copyButtonProps is the properties for the CopyButton component.
type copyButtonProps struct {
	// TextToCopy is the text that should be copied to the
	// clipboard when the button is clicked.
	TextToCopy string
	// Size is the size of the button. Can be "sm", "md" (default) or "lg".
	Size size
}

// copyButton is a button that copies text to the clipboard when clicked.
//
// The text is stored in a data-copy attribute (HTML-escaped by nodx) and read
// by one static onclick handler. Never build JavaScript source from the text:
// it can be a stored value such as a connection string or an S3 key.
func copyButton(props copyButtonProps) nodx.Node {
	return nodx.Div(
		nodx.Class("inline-block tooltip tooltip-right"),
		nodx.Data("tip", "Copy to clipboard"),
		nodx.Button(
			nodx.ClassMap{
				"btn btn-neutral btn-square btn-ghost": true,
				"btn-sm":                               props.Size == SizeSm,
				"btn-lg":                               props.Size == SizeLg,
			},
			nodx.Type("button"),
			nodx.TitleAttr("Copy to clipboard"),
			nodx.Data("copy", props.TextToCopy),
			nodx.Attr("onclick", "copyToClipboard(this.dataset.copy)"),
			lucide.Copy(),
		),
	)
}
