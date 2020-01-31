package curl

const (
	UAP_DEFAULT = 0
	UAP_HUMAN   // firefox, chrome ...
	UAP_RANDOM
)

var (
	human_uas = []string{
		// "firefox", "chrome", "IE", "Safari",
		"Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0",
	}
)

func rand_humanua() string {
	ualen := human_uas.len()
	return human_uas[ualen-1]
}

func rand_randomua() string {
	return ""
}
