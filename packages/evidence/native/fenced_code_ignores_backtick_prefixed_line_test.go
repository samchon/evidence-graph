package evidence

import "testing"

// Verifies that a code line beginning with the fence character but carrying
// trailing text does not close the block.
//
// fenceMarker matched the leading backtick run and ignored the rest, so a line
// like ```stop inside a block closed it early: the code after it leaked as
// headings, and the block's real closing fence then re-opened one, dropping
// every heading later in the file. CommonMark closes a fence only with a pure
// run of the fence character.
//
//  1. Put a backtick-prefixed content line and a fake heading inside a fence.
//  2. Close the block with a pure fence and declare another real heading.
//  3. Assert only the headings outside the fence are indexed.
func TestFencedCodeIgnoresBacktickPrefixedLine(t *testing.T) {
	content := "## Shell Demo\n" +
		"\n" +
		"```bash\n" +
		"echo hi\n" +
		"```stop\n" +
		"# Not Real\n" +
		"done\n" +
		"```\n" +
		"\n" +
		"## After The Block\n"

	got := map[string]bool{}
	for _, section := range scanMarkdownSections(content) {
		got[section.Anchor] = true
	}
	if got["not-real"] {
		t.Fatal("a heading inside a code block leaked as a section (fence closed early)")
	}
	if !got["shell-demo"] {
		t.Fatal("the heading before the block went missing")
	}
	if !got["after-the-block"] {
		t.Fatal("a heading after the block was dropped (fence parity desynced)")
	}
}

// Verifies that four-space-indented fence runs cannot open or close a fenced
// code block.
//
// CommonMark treats a four-space-indented fence run as ordinary indented code.
// Accepting one as a fence flips the scanner's state: an apparent opener hides
// later real headings, while an apparent closer leaks code headings and makes
// the real closer open a second block.
//
//  1. Put a real heading after a four-space-indented apparent opener.
//  2. Put a fake heading after a four-space-indented apparent closer.
//  3. Assert the real heading is indexed and the fenced fake stays hidden.
func TestFencedCodeRequiresAtMostThreeLeadingSpaces(t *testing.T) {
	apparentOpener := "## Before Indented Code\n" +
		"\n" +
		"    ```bash\n" +
		"## After Indented Code\n"
	got := anchorsOf(apparentOpener)
	if !got["after-indented-code"] {
		t.Fatal("a four-space-indented apparent opener hid a later real heading")
	}

	apparentCloser := "## Before The Block\n" +
		"\n" +
		"```text\n" +
		"    ```\n" +
		"# Not Real\n" +
		"```\n" +
		"\n" +
		"## After The Block\n"
	got = anchorsOf(apparentCloser)
	if got["not-real"] {
		t.Fatal("a four-space-indented apparent closer leaked a code heading")
	}
	if !got["after-the-block"] {
		t.Fatal("the real closer failed to restore heading scanning")
	}
}

func anchorsOf(content string) map[string]bool {
	got := map[string]bool{}
	for _, section := range scanMarkdownSections(content) {
		got[section.Anchor] = true
	}
	return got
}
