/*
 * Revision History:
 *     Initial: 2018/06/02        Feng Yifei
 */

package main

import (
	"fmt"
)

func main() {
	s := []int{}
	fmt.Println(len(s), cap(s))

	for i := 0; i < 128; i++ {
		s = append(s, i)
		fmt.Println(len(s), cap(s))
	}
}
