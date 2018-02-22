/*
 * MIT License
 *
 * Copyright (c) 2018 TechCatsLab
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

/*
 * Revision History:
 *     Initial: 2018/02/22        Feng Yifei
 */

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"
)

// Packet .
type Packet struct {
	ID    int16
	Value uint16
}

// Marshal .
func (p *Packet) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, p.ID)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, p.Value)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Unmarshal .
func (p *Packet) Unmarshal(bin []byte) error {
	buf := bytes.NewBuffer(bin)

	err := binary.Read(buf, binary.BigEndian, &p.ID)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &p.Value)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	packet := &Packet{
		ID:    1,
		Value: 2,
	}

	fmt.Println("size:", unsafe.Sizeof(*packet))

	bin, err := packet.Marshal()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(bin)

	err = packet.Unmarshal(bin)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(packet)
}
