package code

import "fmt"

var base []string = []string{"2", "3", "4", "5", "6", "7", "8", "9",
	"a", "A", "b", "B", "c", "C", "d", "D", "e", "E", "f", "F", "g", "G", "h", "H", "j", "J",
	"k", "K", "m", "M", "n", "N", "p", "P", "q", "Q", "r", "R", "s", "S", "t", "T", "u", "U",
	"v", "V", "w", "W", "x", "X", "y", "Y", "z", "Z"}

func Code(id int64) string {
	return code(id, "")
}

func code(id int64, cc string) string {
	x := id % 54
	n := base[x]
	id = id / 54

	if id == 0 {
		return fmt.Sprintf("%s%s", n, cc)
	} else {
		return fmt.Sprintf("%s%s%s", code(id, cc), n, cc)
	}
}
