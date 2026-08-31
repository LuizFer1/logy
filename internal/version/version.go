package version

// Set via -ldflags at release build time.
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

func String() string {
	return format(Version, Commit, Date)
}

func format(v, commit, date string) string {
	if commit == "" && date == "" {
		return v
	}
	extra := commit
	if date != "" {
		if extra != "" {
			extra += " "
		}
		extra += date
	}
	return v + " (" + extra + ")"
}
