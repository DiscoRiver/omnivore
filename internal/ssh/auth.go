package ssh

var (
	AuthType = ""
)

func authTypeIsSet() bool {
	if AuthType != "" {
		return true
	}
	return false
}
