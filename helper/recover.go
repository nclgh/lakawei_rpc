package helper

import (
	"fmt"
	"runtime"
	"strings"
)

type RecoverAfter func(err interface{}, stacks string)

func RecoverPanic(after RecoverAfter) {
	if err := recover(); err != nil {
		stacks := GetStacks(4, 10)
		after(err, strings.Join(stacks, "\n    "))
	}
}

func GetStacks(skip int, maxNum int) []string {
	pc := make([]uintptr, maxNum)
	n := runtime.Callers(skip, pc)

	var stacks []string
	for i := 0; i < n; i ++ {
		f := runtime.FuncForPC(pc[i])
		if f == nil {
			stacks = append(stacks, "unknown Func")
		} else {
			file, line := f.FileLine(pc[i])
			stacks = append(stacks, fmt.Sprintf("%v %v %v", f.Name(), file, line))
		}
	}

	return stacks
}
