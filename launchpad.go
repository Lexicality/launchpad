// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package launchpad

import (
	"errors"
	"strings"
	"time"

	"github.com/rakyll/portmidi"
)

type Launchpad struct {
	inputStream  *portmidi.Stream
	outputStream *portmidi.Stream
}

type Hit struct {
	X int
	Y int
}

func New() (*Launchpad, error) {
	input, output, err := discover()
	if err != nil {
		return nil, err
	}

	var inStream, outStream *portmidi.Stream
	if inStream, err = portmidi.NewInputStream(input, 1024); err != nil {
		return nil, err
	}
	if outStream, err = portmidi.NewOutputStream(output, 1024, 0); err != nil {
		return nil, err
	}
	return &Launchpad{inputStream: inStream, outputStream: outStream}, nil
}

func (l *Launchpad) Listen() <-chan Hit {
	ch := make(chan Hit)
	go func(pad *Launchpad, ch chan Hit) {
		for {
			// sleep for a while before the new polling tick,
			// otherwise operation is too intensive and blocking
			time.Sleep(10 * time.Millisecond)
			hits, err := pad.Read()
			if err != nil {
				continue
			}
			for i := range hits {
				ch <- hits[i]
			}
		}
	}(l, ch)
	return ch
}

func (l *Launchpad) Read() (hits []Hit, err error) {
	var evts []portmidi.Event
	if evts, err = l.inputStream.Read(64); err != nil {
		return
	}
	for _, evt := range evts {
		if evt.Data1 > 0 {
			var x, y int64
			x = evt.Status % 8
			y = 7 - ((evt.Status - x) / 16)
			hits = append(hits, Hit{X: int(x), Y: int(y)})
		}
	}
	return
}

func (l *Launchpad) Light(x, y, g, r int) error {
	note := int64(x + 16*(7-y))
	velocity := int64(16*g + r + 8 + 4)
	return l.outputStream.WriteShort(0x90, note, velocity)
}

func (l *Launchpad) Reset() error {
	return l.outputStream.WriteShort(0xb0, 0, 0)
}

func (l *Launchpad) Cleanup() error {
	if err := l.inputStream.Close(); err != nil {
		return err
	}
	return l.outputStream.Close()
}

func discover() (input portmidi.DeviceId, output portmidi.DeviceId, err error) {
	in := -1
	out := -1
	for i := 0; i < portmidi.CountDevices(); i++ {
		info := portmidi.GetDeviceInfo(portmidi.DeviceId(i))
		if strings.Contains(info.Name, "Launchpad") {
			if info.IsInputAvailable {
				in = i
			}
			if info.IsOutputAvailable {
				out = i
			}
		}
	}
	if in == -1 || out == -1 {
		err = errors.New("portmidi: No Launchpad is connected.")
	} else {
		input = portmidi.DeviceId(in)
		output = portmidi.DeviceId(out)
	}
	return
}
