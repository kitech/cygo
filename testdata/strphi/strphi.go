package main

func main() {
	env := "foo,bar,baz"
	for env != "" {
		i := -1
		var field string
		field, env = env[:i], env[i+1:]
		println(env, field)
	}
}
