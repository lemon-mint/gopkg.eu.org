package types

type Module struct {
	Path string // FS Path relative to SOURCE_PATH

	Root    string // required
	VCS     string // default: git
	RepoURL string // required

	Description string // optional
}
