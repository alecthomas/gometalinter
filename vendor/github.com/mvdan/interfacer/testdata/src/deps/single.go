package single

func BasicWrong(rc ReadCloser) { // WARN rc can be Closer
	rc.Close()
}
