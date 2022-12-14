// shamelessly copy/pasted from github.com/hybridgroup/gobot/sysfs/digital_pin.go
package gpio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
	"time"
)

const (
	// IN gpio direction
	IN = "in"
	// OUT gpio direction
	OUT = "out"
	// HIGH gpio level
	HIGH = 1
	// LOW gpio level
	LOW = 0
	// GPIOPATH default linux gpio path
	GPIOPATH = "/sys/class/gpio"
)

var errNotExported = errors.New("pin has not been exported")

type DigitalPin struct {
	pin   string
	label string

	value     *os.File
	direction *os.File
}

// NewDigitalPin returns a DigitalPin given the pin number
func NewDigitalPin(pin int) *DigitalPin {
	d := &DigitalPin{pin: strconv.Itoa(pin)}
	d.label = "gpio" + d.pin

	return d
}

// Direction sets the direction for the pin
func (d *DigitalPin) Direction(dir string) error {
	_, err := writeFile(d.direction, []byte(dir))
	return err
}

// Write writes to the pin
func (d *DigitalPin) Write(b int) error {
	_, err := writeFile(d.value, []byte(strconv.Itoa(b)))
	return err
}

// Read reads the current value of the pin
func (d *DigitalPin) Read() (n int, err error) {
	buf, err := readFile(d.value)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(buf[0]))
}

// Export exports the pin for use by the operating system
func (d *DigitalPin) Export() error {
	export, err := os.OpenFile(GPIOPATH+"/export", os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer export.Close()

	_, err = writeFile(export, []byte(d.pin))
	if err != nil {
		// If EBUSY then the pin has already been exported
		e, ok := err.(*os.PathError)
		if !ok || e.Err != syscall.EBUSY {
			return err
		}
	}

	if d.direction != nil {
		d.direction.Close()
	}

	attempt := 0
	for {
		attempt++
		d.direction, err = os.OpenFile(fmt.Sprintf("%v/%v/direction", GPIOPATH, d.label), os.O_RDWR, 0644)
		if err == nil {
			break
		}
		if attempt > 10 {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}

	if d.value != nil {
		d.value.Close()
	}
	if err == nil {
		d.value, err = os.OpenFile(fmt.Sprintf("%v/%v/value", GPIOPATH, d.label), os.O_RDWR, 0644)
	}

	if err != nil {
		// Should we unexport here?
		// If we don't unexport we should make sure to close d.direction and d.value here
		d.Unexport()
	}

	return err
}

// Unexport unexports the pin and releases the pin from the operating system
func (d *DigitalPin) Unexport() error {
	unexport, err := os.OpenFile(GPIOPATH+"/unexport", os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer unexport.Close()

	if d.direction != nil {
		d.direction.Close()
		d.direction = nil
	}
	if d.value != nil {
		d.value.Close()
		d.value = nil
	}

	_, err = writeFile(unexport, []byte(d.pin))
	if err != nil {
		// If EINVAL then the pin is reserved in the system and can't be unexported
		e, ok := err.(*os.PathError)
		if !ok || e.Err != syscall.EINVAL {
			return err
		}
	}

	return nil
}

// Linux sysfs / GPIO specific sysfs docs.
//  https://www.kernel.org/doc/Documentation/filesystems/sysfs.txt
//  https://www.kernel.org/doc/Documentation/gpio/sysfs.txt

var writeFile = func(f *os.File, data []byte) (i int, err error) {
	if f == nil {
		return 0, errNotExported
	}

	// sysfs docs say:
	// > When writing sysfs files, userspace processes should first read the
	// > entire file, modify the values it wishes to change, then write the
	// > entire buffer back.
	// however, this seems outdated/inaccurate (docs are from back in the Kernel BitKeeper days).

	i, err = f.Write(data)
	return i, err
}

var readFile = func(f *os.File) ([]byte, error) {
	if f == nil {
		return nil, errNotExported
	}

	// sysfs docs say:
	// > If userspace seeks back to zero or does a pread(2) with an offset of '0' the [..] method will
	// > be called again, rearmed, to fill the buffer.

	// TODO: Examine if seek is needed if full buffer is read from sysfs file.

	buf := make([]byte, 2)
	_, err := f.Seek(0, io.SeekStart)
	if err == nil {
		_, err = f.Read(buf)
	}
	return buf, err
}
