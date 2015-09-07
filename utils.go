package irc

func prepend(e interface{}, v []interface{}) []interface{} {
	var vc []interface{}

	vc = append(vc, e)
	vc = append(vc, v...)

	return vc
}
