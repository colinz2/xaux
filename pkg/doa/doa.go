package doa

func MustTrue(m bool, msg string) {
	if m == false {
		panic(msg)
	}
}
